package ingress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	echo "github.com/labstack/echo/v4"
	"github.com/pion/rtcp"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/pkg/controller/ingress"
	globalprotocol "github.com/romashorodok/conferencing-platform/pkg/protocol"
	"go.uber.org/atomic"
	"go.uber.org/fx"
)

/* Utils start */

// ParallelExec will executes the given function with each element of vals, if len(vals) >= parallelThreshold,
// will execute them in parallel, with the given step size. So fn must be thread-safe.
func ParallelExec[T any](vals []T, parallelThreshold, step uint64, fn func(T)) {
	if uint64(len(vals)) < parallelThreshold {
		for _, v := range vals {
			fn(v)
		}
		return
	}

	// parallel - enables much more efficient multi-core utilization
	start := atomic.NewUint64(0)
	end := uint64(len(vals))

	var wg sync.WaitGroup
	numCPU := runtime.NumCPU()
	wg.Add(numCPU)
	for p := 0; p < numCPU; p++ {
		go func() {
			defer wg.Done()
			for {
				n := start.Add(step)
				if n >= end+step {
					return
				}

				for i := n - step; i < n && i < end; i++ {
					fn(vals[i])
				}
			}
		}()
	}
	wg.Wait()
}

/* Utils end */

var INGEST_ANSWER_TYPE = "answer"

type whipController struct {
	roomService protocol.RoomService
	logger      *slog.Logger
	webrtc      *webrtc.API
}

// WebrtcHttpIngestionControllerWebrtcHttpIngest implements ingress.ServerInterface.
func (*whipController) WebrtcHttpIngestionControllerWebrtcHttpIngest(ctx echo.Context, sessionID string) error {
	panic("unimplemented")
}

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

type threadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

func (t *threadSafeWriter) WriteJSON(val interface{}) error {
	t.Lock()
	defer t.Unlock()

	return t.Conn.WriteJSON(val)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Publisher struct {
	peerConnection *webrtc.PeerConnection
}

func NewPublisher(peerConnection *webrtc.PeerConnection) (*Publisher, error) {
	return &Publisher{
		peerConnection: peerConnection,
	}, nil
}

type PeerContext struct {
	// publisher  *webrtc.PeerConnection
	publisher  *Publisher
	subscriber *Subscriber
	// subscriber          *webrtc.PeerConnection
	negotiationDataChan *webrtc.DataChannel
}

type SdpDescription struct {
	Type string `json:"type"`
	Sdp  string `json:"sdp"`
}

type TrackContext struct {
	track  *webrtc.TrackLocalStaticRTP
	sender *webrtc.RTPSender
}

func (t *TrackContext) Close() (err error) {
	err = t.sender.Stop()
	err = t.sender.Transport().Stop()
	err = t.sender.ReplaceTrack(nil)
	return
}

type LoopbackTrackContext struct {
	TrackContext
	transceiver *webrtc.RTPTransceiver
}

var subscribers = make(map[string]*Subscriber)

type Subscriber struct {
	peerConnection *webrtc.PeerConnection
	id             string

	sid string

	loopback map[string]*LoopbackTrackContext
	tracks   map[string]*TrackContext
	tracksMu sync.Mutex
}

func (s *Subscriber) Close() (err error) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	for id, t := range s.tracks {
		err = t.Close()
		log.Println("close track error", err)
		err = s.peerConnection.RemoveTrack(t.sender)
		log.Println("remove track error", err)
		delete(s.tracks, id)
	}

	for id, t := range s.loopback {
		err = t.Close()
		log.Println("close track error", err)
		err = s.peerConnection.RemoveTrack(t.sender)
		log.Println("remove track error", err)
		delete(s.loopback, id)
	}

	return err
}

func (s *Subscriber) HasTrack(trackID string) (*TrackContext, bool) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()
	track, exist := s.tracks[trackID]
	return track, exist
}

// May return already existed track if it has race condition
func (s *Subscriber) CreateTrack(t *webrtc.TrackRemote) (*TrackContext, error) {
	// NOTE: Track may have same id, but it may have different layerID(RID)

	track, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		log.Println("unable create track for subscriber. track", err)
		return nil, err
	}

	sender, err := s.peerConnection.AddTrack(track)
	if err != nil {
		log.Println("uanble add track to the subscriber. sender", err)
		return nil, err
	}

	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	if track, exist := s.tracks[t.ID()]; exist {
		if track != nil {
			return track, nil
		}
	}

	trackContext := &TrackContext{track: track, sender: sender}
	s.tracks[t.ID()] = trackContext
	return trackContext, nil
}

func (s *Subscriber) DeleteTrack(trackID string) error {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()
	if _, exist := s.tracks[trackID]; !exist {
		return errors.New("Track not exist. Unable delete")
	}
	delete(s.tracks, trackID)
	return nil
}

func (s *Subscriber) LoopbackTrack(id string, sid string, capability webrtc.RTPCodecCapability) error {
	track, err := webrtc.NewTrackLocalStaticRTP(capability, id, sid)
	if err != nil {
		return err
	}

	transceiver, err := s.peerConnection.AddTransceiverFromTrack(track, webrtc.RTPTransceiverInit{
		// Recvonly not supported
		Direction: webrtc.RTPTransceiverDirectionSendrecv,
	})
	if err != nil {
		return err
	}
	// Disable loopback
	// transceiver.SetSender(transceiver.Sender(), nil)

	s.loopback[id] = &LoopbackTrackContext{
		TrackContext: TrackContext{
			track:  track,
			sender: transceiver.Sender(),
		},
		transceiver: transceiver,
	}
	return nil
}

func NewSubscriber(peerConnection *webrtc.PeerConnection) (*Subscriber, error) {
	return &Subscriber{
		id:             uuid.New().String(),
		peerConnection: peerConnection,
		loopback:       make(map[string]*LoopbackTrackContext),
		tracks:         make(map[string]*TrackContext),
	}, nil
}

type SubscriberPool struct {
	subscriberMu sync.Mutex

	pool map[string]*Subscriber
}

func (s *SubscriberPool) Get() []*Subscriber {
	s.subscriberMu.Lock()
	defer s.subscriberMu.Unlock()

	var result []*Subscriber
	for _, sub := range s.pool {
		result = append(result, sub)
	}
	return result
}

func (s *SubscriberPool) Add(sub *Subscriber) error {
	s.subscriberMu.Lock()
	defer s.subscriberMu.Unlock()

	if _, exist := s.pool[sub.id]; exist {
		return errors.New("Subscriber exist. Remove it first")
	}

	s.pool[sub.id] = sub
	return nil
}

func (s *SubscriberPool) Remove(sub *Subscriber) {
	s.subscriberMu.Lock()
	defer s.subscriberMu.Unlock()
	if sub == nil {
		return
	}
	if s, exist := s.pool[sub.id]; exist {
		_ = s.Close()
	}
	delete(s.pool, sub.id)
}

func NewSubscriberPool() *SubscriberPool {
	return &SubscriberPool{
		pool: make(map[string]*Subscriber),
	}
}

var subscriberPool = NewSubscriberPool()

type SdpAnswer struct {
	Type string `json:"type"`
	Sdp  string `json:"sdp"`
}

func (ctrl *whipController) WebrtcHttpIngestionControllerWebsocketRtcSignal(ctx echo.Context) error {
	conn, err := upgrader.Upgrade(ctx.Response().Writer, ctx.Request(), nil)
	if err != nil {
		ctrl.logger.Error(fmt.Sprintf("Unable upgrade request %s", ctx.Request()))
		return err
	}

	w := &threadSafeWriter{conn, sync.Mutex{}}
	defer conn.Close()

	peerContext := &PeerContext{}
	defer func() {
		if peerContext.subscriber != nil {
			_ = peerContext.subscriber.peerConnection.Close()
		}
	}()

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Println("Unable create subscriber peer")
		return err
	}

	subscriberConnCtx, subscriberConnCtxCancel := context.WithCancelCause(context.TODO())
	subscriber, _ := NewSubscriber(peerConnection)
	subscriberPool.Add(subscriber)
	defer subscriberPool.Remove(peerContext.subscriber)
	defer subscriberConnCtxCancel(errors.New("Defer implicit connection close"))

	peerContext.subscriber = subscriber

	// Declare publisher transceiver for all kinds
	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := subscriber.peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Print(err)
			return err
		}
	}

	// Need send ice candidate to the client for success gathering
	peerContext.subscriber.peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		log.Println(i)
		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			log.Println(err)
			return
		}
		w.WriteJSON(&websocketMessage{
			Event: "candidate",
			Data:  string(candidateString),
		})
	})

	peerContext.subscriber.peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		// Create a track to fan out our incoming video to all peers
		// trackLocal := addTrack(t)
		// defer removeTrack(trackLocal)
		//
		// buf := make([]byte, 1500)
		// for {
		// 	i, _, err := t.Read(buf)
		// 	if err != nil {
		// 		return
		// 	}
		//
		// 	if _, err = trackLocal.Write(buf[:i]); err != nil {
		// 		return
		// 	}
		// }

		defer func() {
			for _, subs := range subscriberPool.Get() {
				_ = subs.DeleteTrack(t.ID())
			}
		}()

		var threshold uint64 = 1000000
		var step uint64 = 2
		log.Println("On track", t.ID())

		for {
			select {
			case <-subscriberConnCtx.Done():
				return
			default:
			}

			pkt, _, err := t.ReadRTP()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				continue
			}

			subscribers := subscriberPool.Get()

			ParallelExec(subscribers, threshold, step, func(sub *Subscriber) {
				select {
				case <-subscriberConnCtx.Done():
					return
				default:
				}

				track, exist := sub.HasTrack(t.ID())
				if !exist {
					// If subscriber is done remove subscriber
					track, err = sub.CreateTrack(t)
					if err != nil {
						return
					}
					signalPeerConnections()
				}

				// WriteRTP takes about 50µs
				track.track.WriteRTP(pkt)
			})
		}
	})

	peerContext.subscriber.peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := peerContext.subscriber.peerConnection.Close(); err != nil {
				log.Print(err)
			}
		case webrtc.PeerConnectionStateClosed:
			signalPeerConnections()
		}
	})

	listLock.Lock()
	peerConnections = append(peerConnections, &peerConnectionState{
		peerContext:    peerContext,
		peerConnection: peerConnection,
		signaling:      &webrtc.DataChannel{},
		ws:             w,
	})
	listLock.Unlock()

	message := &websocketMessage{}
	for {
		if err := conn.ReadJSON(message); err != nil {
			ctrl.logger.Error(fmt.Sprintf("%s wrong format support only JSON", conn.RemoteAddr()))
			w.WriteJSON(&websocketMessage{
				Event: "error",
				Data:  "wrong data format",
			})
			return err
		}
		switch message.Event {
		case "offer":
			// The offer must send a publisher peer
			{
			}
		case "answer":
			// The answer should be from subscriber peer
			{
				var answer SdpAnswer
				if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
					log.Println("at sdp", err)
					return err
				}

				if err := peerContext.subscriber.peerConnection.SetRemoteDescription(webrtc.SessionDescription{
					Type: webrtc.SDPTypeAnswer,
					SDP:  answer.Sdp,
				}); err != nil {
					log.Println("Set remote descriptor for subscriber fail", err)
					return err
				}

				log.Println("All done")
			}

		case "subscribe":
			{
				sid := uuid.NewString()

				if err = subscriber.LoopbackTrack(uuid.NewString(), sid, webrtc.RTPCodecCapability{
					MimeType: webrtc.MimeTypeVP8,
				}); err != nil {
					log.Println(err)
					return err
				}

				if err = subscriber.LoopbackTrack(uuid.NewString(), sid, webrtc.RTPCodecCapability{
					MimeType: webrtc.MimeTypeOpus,
				}); err != nil {
					log.Println(err)
					return err
				}

				dc, err := subscriber.peerConnection.CreateDataChannel("_negotiation", nil)
				if err != nil {
					log.Println(err)
					return err
				}
				peerContext.negotiationDataChan = dc

				offer, err := subscriber.peerConnection.CreateOffer(nil)
				if err != nil {
					log.Println(err)
					return err
				}
				if err = subscriber.peerConnection.SetLocalDescription(offer); err != nil {
					log.Println(err)
					return err
				}

				offerJSON, err := json.Marshal(offer)
				if err != nil {
					log.Println(err)
					return err
				}

				if err = w.WriteJSON(&websocketMessage{
					Event: "offer",
					Data:  string(offerJSON),
				}); err != nil {
					log.Println(err)
					return err
				}

			}
		}
		log.Println("PeerContext", peerContext)
	}
}

const (
	MISSING_ICE_UFRAG_MSG = "offer must contain at least one track or data channel"
	NOT_FOUND_ROOM_MSG    = "not found room"
)

var (
	listLock        sync.RWMutex
	peerConnections []*peerConnectionState
	trackLocals     = make(map[string]*webrtc.TrackLocalStaticRTP)
)

type peerConnectionState struct {
	peerContext    *PeerContext
	peerConnection *webrtc.PeerConnection
	signaling      *webrtc.DataChannel
	ws             *threadSafeWriter
}

// Add to list of tracks and fire renegotation for all PeerConnections
func addTrack(t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		signalPeerConnections()
	}()

	// Create a new TrackLocal with the same codec as our incoming
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	trackLocals[t.ID()] = trackLocal
	return trackLocal
}

// Remove from list of tracks and fire renegotation for all PeerConnections
func removeTrack(t *webrtc.TrackLocalStaticRTP) {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		signalPeerConnections()
	}()

	delete(trackLocals, t.ID())
}

func signalPeerConnections() {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		dispatchKeyFrame()
	}()
	attemptSync := func() (tryAgain bool) {
		for _, conn := range peerConnections {
			log.Printf("sync attempt %+v", conn.peerContext.subscriber)

			if conn.peerContext.subscriber.peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				continue
			}

			// existingSenders := map[string]bool{}

			// // Clean prev senders
			// for _, sender := range conn.peerContext.subscriber.peerConnection.GetSenders() {
			// 	if sender.Track() == nil {
			// 		continue
			// 	}
			// 	// existingSenders[sender.Track().ID()] = true
			//
			// 	if _, ok := trackLocals[sender.Track().ID()]; !ok {
			// 		if err := conn.peerContext.subscriber.peerConnection.RemoveTrack(sender); err != nil {
			// 			return true
			// 		}
			// 	}
			// }

			// // Don't add current publisher senders to subscriber to receive loopback
			// for _, receiver := range conn.peerContext.subscriber.peerConnection.GetReceivers() {
			// 	if receiver.Track() == nil {
			// 		continue
			// 	}
			// 	existingSenders[receiver.Track().ID()] = true
			// }

			// for _, track := range conn.peerContext.subscriber.tracks {
			// 	_, _ = conn.peerConnection.AddTransceiverFromTrack(track.track, webrtc.RTPTransceiverInit{
			// 		Direction: webrtc.RTPTransceiverDirectionSendonly,
			// 	})
			// }

			// log.Println(existingSenders)

			// for trackID := range trackLocals {
			// 	if _, exists := existingSenders[trackID]; !exists {
			// 		// log.Println(trackLocals[trackID].ID(), trackLocals[trackID].StreamID())
			// 		if _, err := peerConnections[i].peerContext.subscriber.AddTrack(trackLocals[trackID]); err != nil {
			// 			return true
			// 		}
			// 	}
			// }

			// you can't break the flow of a new RTCPeerConnections (create offer -> set local -> set remote -> create answer -> set local -> set remote). If you create offer on one side, answer on the other and only then set the local/remote descriptions it should break by design.

			// offer, err := peerConnections[i].peerContext.subscriber.CreateOffer(nil)
			offer, err := conn.peerContext.subscriber.peerConnection.CreateOffer(nil)
			if err != nil {
				log.Println("offer error", err)
				return true
			}

			if err = conn.peerContext.subscriber.peerConnection.SetLocalDescription(offer); err != nil {
				log.Println("local desc", err)
				return true
			}

			log.Println("Dispatch offer")

			//
			offerString, err := json.Marshal(offer)
			if err != nil {
				return true
			}
			//
			if err = conn.ws.WriteJSON(&websocketMessage{
				Event: "offer",
				Data:  string(offerString),
			}); err != nil {
				return true
			}
		}
		return
	}

	// attemptSync := func() (tryAgain bool) {
	// 	for i := range peerConnections {
	// 		if peerConnections[i].peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
	// 			peerConnections = append(peerConnections[:i], peerConnections[i+1:]...)
	// 			return true // We modified the slice, start from the beginning
	// 		}
	//
	// 		if peerConnections[i].peerConnection.ConnectionState() != webrtc.PeerConnectionStateConnected {
	// 			return true
	// 		}
	//
	// 		// map of sender we already are seanding, so we don't double send
	// 		existingSenders := map[string]bool{}
	//
	// 		for _, sender := range peerConnections[i].peerConnection.GetSenders() {
	// 			if sender.Track() == nil {
	// 				continue
	// 			}
	//
	// 			existingSenders[sender.Track().ID()] = true
	//
	// 			// If we have a RTPSender that doesn't map to a existing track remove and signal
	// 			if _, ok := trackLocals[sender.Track().ID()]; !ok {
	// 				if err := peerConnections[i].peerConnection.RemoveTrack(sender); err != nil {
	// 					return true
	// 				}
	// 			}
	// 		}
	// 		// log.Println(peerConnections[i].peerConnection.GetReceivers())
	//
	// 		// Don't receive videos we are sending, make sure we don't have loopback
	// 		for _, receiver := range peerConnections[i].peerConnection.GetReceivers() {
	// 			if receiver.Track() == nil {
	// 				continue
	// 			}
	//
	// 			existingSenders[receiver.Track().ID()] = true
	// 		}
	//
	// 		// Add all track we aren't sending yet to the PeerConnection
	// 		for trackID := range trackLocals {
	// 			if _, ok := existingSenders[trackID]; !ok {
	// 				if _, err := peerConnections[i].peerConnection.AddTrack(trackLocals[trackID]); err != nil {
	// 					return true
	// 				}
	// 			}
	// 		}
	//
	// 		<-webrtc.GatheringCompletePromise(peerConnections[i].peerConnection)
	//
	// 		offer, err := peerConnections[i].peerConnection.CreateOffer(nil)
	// 		if err != nil {
	// 			log.Println("offer error", err)
	// 			return true
	// 		}
	//
	// 		if err = peerConnections[i].peerConnection.SetLocalDescription(offer); err != nil {
	// 			log.Println("local desc", err)
	// 			return true
	// 		}
	//
	// 		log.Println("Dispatch offer")
	//
	// 		offerString, err := json.Marshal(offer)
	// 		if err != nil {
	// 			return true
	// 		}
	//
	// 		if err = peerConnections[i].ws.WriteJSON(&websocketMessage{
	// 			Event: "offer",
	// 			Data:  string(offerString),
	// 		}); err != nil {
	// 			return true
	// 		}
	// 	}
	//
	// 	return
	// }

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
			go func() {
				time.Sleep(time.Second * 3)
				signalPeerConnections()
			}()
			return
		}

		if !attemptSync() {
			break
		}
	}
}

// dispatchKeyFrame sends a keyframe to all PeerConnections, used everytime a new user joins the call
func dispatchKeyFrame() {
	listLock.Lock()
	defer listLock.Unlock()

	for i := range peerConnections {
		for _, receiver := range peerConnections[i].peerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = peerConnections[i].peerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}

// signalPeerConnections updates each PeerConnection so that it is getting all the expected media tracks
// func signalPeerConnections() {
// 	listLock.Lock()
// 	defer func() {
// 		listLock.Unlock()
// 		dispatchKeyFrame()
// 	}()
//
// 	attemptSync := func() (tryAgain bool) {
// 		for i := range peerConnections {
// 			if peerConnections[i].peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
// 				peerConnections = append(peerConnections[:i], peerConnections[i+1:]...)
// 				return true // We modified the slice, start from the beginning
// 			}
//
// 			// map of sender we already are seanding, so we don't double send
// 			existingSenders := map[string]bool{}
//
// 			for _, sender := range peerConnections[i].peerConnection.GetSenders() {
// 				if sender.Track() == nil {
// 					continue
// 				}
//
// 				existingSenders[sender.Track().ID()] = true
//
// 				// If we have a RTPSender that doesn't map to a existing track remove and signal
// 				if _, ok := trackLocals[sender.Track().ID()]; !ok {
// 					if err := peerConnections[i].peerConnection.RemoveTrack(sender); err != nil {
// 						return true
// 					}
// 				}
// 			}
// 			log.Println(peerConnections[i].peerConnection.GetReceivers())
//
// 			// Don't receive videos we are sending, make sure we don't have loopback
// 			for _, receiver := range peerConnections[i].peerConnection.GetReceivers() {
// 				if receiver.Track() == nil {
// 					continue
// 				}
//
// 				existingSenders[receiver.Track().ID()] = true
// 			}
//
// 			// Add all track we aren't sending yet to the PeerConnection
// 			for trackID := range trackLocals {
// 				if _, ok := existingSenders[trackID]; !ok {
// 					if _, err := peerConnections[i].peerConnection.AddTrack(trackLocals[trackID]); err != nil {
// 						return true
// 					}
// 				}
// 			}
// 			log.Println(existingSenders)
//
// 			offer, err := peerConnections[i].peerConnection.CreateOffer(nil)
// 			if err != nil {
// 				return true
// 			}
//
// 			<-webrtc.GatheringCompletePromise(peerConnections[i].peerConnection)
// 			// you can't break the flow of a new RTCPeerConnections (create offer -> set local -> set remote -> create answer -> set local -> set remote). If you create offer on one side, answer on the other and only then set the local/remote descriptions it should break by design.
//
// 			if err = peerConnections[i].peerConnection.SetLocalDescription(offer); err != nil {
// 				return true
// 			}
//
// 			offerString, err := json.Marshal(offer)
// 			if err != nil {
// 				return true
// 			}
// 			if err = peerConnections[i].signaling.SendText(string(offerString)); err != nil {
// 				return true
// 			}
// 		}
//
// 		return
// 	}
//
// 	for syncAttempt := 0; ; syncAttempt++ {
// 		if syncAttempt == 25 {
// 			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
// 			go func() {
// 				time.Sleep(time.Second * 3)
// 				signalPeerConnections()
// 			}()
// 			log.Println("Failed sync")
// 			return
// 		}
//
// 		if !attemptSync() {
// 			log.Println("success sync")
// 			break
// 		}
// 	}
// }
//
// // dispatchKeyFrame sends a keyframe to all PeerConnections, used everytime a new user joins the call
// func dispatchKeyFrame() {
// 	listLock.Lock()
// 	defer listLock.Unlock()
//
// 	for i := range peerConnections {
// 		for _, receiver := range peerConnections[i].peerConnection.GetReceivers() {
// 			if receiver.Track() == nil {
// 				continue
// 			}
//
// 			_ = peerConnections[i].peerConnection.WriteRTCP([]rtcp.Packet{
// 				&rtcp.PictureLossIndication{
// 					MediaSSRC: uint32(receiver.Track().SSRC()),
// 				},
// 			})
// 		}
// 	}
// }

// func (ctrl *whipController) WebrtcHttpIngestionControllerWebrtcHttpIngest(ctx echo.Context, sessionID string) error {
// 	var request ingress.WebrtcHttpIngestRequest
//
// 	if err := json.NewDecoder(ctx.Request().Body).Decode(&request); err != nil {
// 		return echo.NewHTTPError(http.StatusInternalServerError, err)
// 	}
// 	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
// 	if err != nil {
// 		log.Print("peerConnection", err)
// 		return err
// 	}
//
// 	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
// 		if _, err := peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
// 			Direction: webrtc.RTPTransceiverDirectionSendrecv,
// 		}); err != nil {
// 			log.Print("transivers errors", err)
// 			return err
// 		}
// 	}
//
// 	peerConnectionDataChannel := &peerConnectionState{
// 		peerConnection: peerConnection,
// 		signaling:      nil,
// 	}
//
// 	listLock.Lock()
// 	peerConnections = append(peerConnections, peerConnectionDataChannel)
// 	listLock.Unlock()
//
// 	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
// 		peerConnectionDataChannel.signaling = dc
//
// 		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
// 			type localPeerAnswer struct {
// 				Type string
// 				Sdp  string
// 			}
//
// 			var answer localPeerAnswer
// 			json.Unmarshal(msg.Data, &answer)
//
// 			desc := webrtc.SessionDescription{
// 				Type: webrtc.SDPTypeAnswer,
// 				SDP:  answer.Sdp,
// 			}
//
// 			if err := peerConnection.SetLocalDescription(desc); err != nil {
// 				log.Println("Unable set answer", err)
// 				return
// 			}
// 			log.Println("Success answer set")
// 		})
// 	})
//
// 	peerConnection.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
// 		trackLocal := addTrack(track)
// 		defer removeTrack(trackLocal)
//
// 		buf := make([]byte, 1500)
// 		for {
// 			i, _, err := track.Read(buf)
// 			if err != nil {
// 				return
// 			}
//
// 			if _, err = trackLocal.Write(buf[:i]); err != nil {
// 				return
// 			}
// 		}
// 	})
//
// 	err = peerConnection.SetRemoteDescription(webrtc.SessionDescription{
// 		Type: webrtc.SDPTypeOffer,
// 		SDP:  *request.Offer.Sdp,
// 	})
// 	if err != nil {
// 		log.Println("Set remote desc", err)
// 		return err
// 	}
//
// 	answer, err := peerConnection.CreateAnswer(nil)
//
// 	peerConnection.SetLocalDescription(answer)
//
// 	<-webrtc.GatheringCompletePromise(peerConnection)
//
// 	ctx.JSON(http.StatusCreated,
// 		&ingress.WebrtcHttpIngestResponse{
// 			Answer: &ingress.SessionDescription{
// 				Sdp:  &peerConnection.LocalDescription().SDP,
// 				Type: &INGEST_ANSWER_TYPE,
// 			},
// 		},
// 	)
//
// 	return nil
// }

// func (ctrl *whipController) WebrtcHttpIngestionControllerWebrtcHttpIngest(ctx echo.Context, sessionID string) error {
// 	var request ingress.WebrtcHttpIngestRequest
//
// 	if err := json.NewDecoder(ctx.Request().Body).Decode(&request); err != nil {
// 		return echo.NewHTTPError(http.StatusInternalServerError, err)
// 	}
//
// 	room := ctrl.roomService.GetRoom(sessionID)
// 	if room == nil {
// 		return echo.NewHTTPError(http.StatusInternalServerError, errors.New(NOT_FOUND_ROOM_MSG))
// 	}
//
// 	peer, err := room.AddParticipant(*request.Offer.Sdp)
// 	if err != nil {
// 		if errors.Is(err, webrtc.ErrSessionDescriptionMissingIceUfrag) {
// 			return echo.NewHTTPError(http.StatusPreconditionFailed, MISSING_ICE_UFRAG_MSG)
// 		}
// 		return echo.NewHTTPError(http.StatusInternalServerError, err)
// 	}
//
// 	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
// 		if _, err := peer.GetPeerConnection().AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
// 			// Direction: webrtc.RTPTransceiverDirectionSendonly,
// 			Direction: webrtc.RTPTransceiverDirectionSendrecv,
// 		}); err != nil {
// 			log.Print(err)
// 			return err
// 		}
// 	}
//
// 	answer, err := peer.GenerateSDPAnswer()
// 	if err != nil {
// 		return echo.NewHTTPError(http.StatusInternalServerError, err)
// 	}
//
// 	ctx.JSON(http.StatusCreated,
// 		&ingress.WebrtcHttpIngestResponse{
// 			Answer: &ingress.SessionDescription{
// 				Sdp:  &answer,
// 				Type: &INGEST_ANSWER_TYPE,
// 			},
// 		},
// 	)
//
// 	return nil
// }

func (*whipController) WebrtcHttpIngestionControllerWebrtcHttpTerminate(ctx echo.Context, sessionID string) error {
	panic("unimplemented")
}

func (ctrl *whipController) Resolve(c *echo.Echo) error {
	spec, err := ingress.GetSwagger()
	if err != nil {
		return err
	}
	spec.Servers = nil
	ingress.RegisterHandlers(c, ctrl)
	return nil
}

var (
	_ ingress.ServerInterface       = (*whipController)(nil)
	_ globalprotocol.HttpResolvable = (*whipController)(nil)
)

type newWhipController_Params struct {
	fx.In

	RoomService protocol.RoomService
	Logger      *slog.Logger
}

func NewWhipController(params newWhipController_Params) *whipController {
	return &whipController{
		roomService: params.RoomService,
		logger:      params.Logger,
	}
}
