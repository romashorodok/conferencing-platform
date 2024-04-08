package ingress

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	echo "github.com/labstack/echo/v4"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/sdp/v3"
	webrtc "github.com/pion/webrtc/v4"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/rtpstats"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/twcc"
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
	// roomService protocol.RoomService
	logger *slog.Logger
	webrtc *webrtc.API
	stats  <-chan *rtpstats.RtpStats

	peerConnectionMu sync.Mutex
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

type PeerContext struct {
	subscriber          *Subscriber
	negotiationDataChan *webrtc.DataChannel
	stats               *rtpstats.RtpStats
	twcc                *twcc.Responder
}

type SdpDescription struct {
	Type string `json:"type"`
	Sdp  string `json:"sdp"`
}

type TrackContextWritable interface {
	WriteRTP(p *rtp.Packet) error
}

type TrackContext struct {
	track  *webrtc.TrackLocalStaticRTP
	sender *webrtc.RTPSender
}

func (t *TrackContext) WriteRTP(p *rtp.Packet) error {
	return t.track.WriteRTP(p)
}

func (t *TrackContext) Close() (err error) {
	err = t.sender.ReplaceTrack(nil)
	err = t.sender.Stop()
	return
}

var _ TrackContextWritable = (*TrackContext)(nil)

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
		err = s.peerConnection.RemoveTrack(t.sender)
		delete(s.tracks, id)
	}

	for id, t := range s.loopback {
		err = t.Close()
		err = s.peerConnection.RemoveTrack(t.sender)
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

func (s *Subscriber) DeleteTrack(trackID string) (err error) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	track, exist := s.tracks[trackID]
	if !exist {
		return errors.New("Track not exist. Unable delete")
	}
	err = track.Close()

	delete(s.tracks, trackID)
	return err
}

func (s *Subscriber) LoopbackTrack(id string, sid string, capability webrtc.RTPCodecCapability) (*LoopbackTrackContext, error) {
	track, err := webrtc.NewTrackLocalStaticRTP(capability, id, sid)
	if err != nil {
		return nil, err
	}

	transceiver, err := s.peerConnection.AddTransceiverFromTrack(track, webrtc.RTPTransceiverInit{
		// Recvonly not supported
		Direction: webrtc.RTPTransceiverDirectionSendonly,
	})
	if err != nil {
		return nil, err
	}

	// NOTE: When new negotiation will be it's will be displayed as video
	// NOTE: To mute track need `webrtc.RTPTransceiverDirectionInactive`
	// Disable loopback sending, or may even replace the track
	// transceiver.SetSender(transceiver.Sender(), nil)

	s.loopback[id] = &LoopbackTrackContext{
		TrackContext: TrackContext{
			track:  track,
			sender: transceiver.Sender(),
		},
		transceiver: transceiver,
	}
	return s.loopback[id], nil
}

func (s *Subscriber) GetLoopbackTrack(trackID string) (track *LoopbackTrackContext, exist bool) {
	track, exist = s.loopback[trackID]
	return
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
	pool         map[string]*Subscriber
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

func (s *SubscriberPool) Remove(sub *Subscriber) (err error) {
	s.subscriberMu.Lock()
	defer s.subscriberMu.Unlock()

	if sub == nil {
		return
	}

	if s, exist := s.pool[sub.id]; exist {
		err = s.Close()
	}

	delete(s.pool, sub.id)
	return err
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

var ErrOnStateClosed error = errors.New("Closed connection")

func debounce(fn func() bool, delay time.Duration) func() bool {
	var (
		mu    sync.Mutex
		last  time.Time
		timer *time.Timer
	)
	return func() (result bool) {
		mu.Lock()
		defer mu.Unlock()

		if timer != nil {
			timer.Stop()
		}

		elapsed := time.Since(last)
		if elapsed > delay {
			result = fn()
			last = time.Now()
			return
		}

		timer = time.AfterFunc(delay-elapsed, func() {
			mu.Lock()
			defer mu.Unlock()
			result = fn()
			last = time.Now()
		})

		return result
	}
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

	log.Println("PeerConnection lock")
	ctrl.peerConnectionMu.Lock()
	peerConnection, err := ctrl.webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Println("Unable create subscriber peer")
		return err
	}
	peerContext.stats = <-ctrl.stats
	ctrl.peerConnectionMu.Unlock()
	log.Println("PeerConnection unlock")

	subscriberConnCtx, subscriberConnCtxCancel := context.WithCancelCause(ctx.Request().Context())
	_ = subscriberConnCtx
	subscriber, err := NewSubscriber(peerConnection)
	if err != nil {
		log.Println("NewSubscriber error")
		return err
	}
	peerContext.subscriber = subscriber
	subscriberPool.Add(subscriber)

	defer subscriberConnCtxCancel(errors.New("Defer implicit connection close"))
	defer subscriberPool.Remove(peerContext.subscriber)

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

	peerContext.subscriber.peerConnection.OnTrack(func(t *webrtc.TrackRemote, recv *webrtc.RTPReceiver) {
		defer func() {
			for _, subs := range subscriberPool.Get() {
				_ = subs.DeleteTrack(t.ID())
			}
		}()

		var threshold uint64 = 1000000
		var step uint64 = 2
		log.Println("On track", t.ID())

		var twccExt uint8
		for _, fb := range t.Codec().RTCPFeedback {
			switch fb.Type {
			case webrtc.TypeRTCPFBGoogREMB:
			case webrtc.TypeRTCPFBNACK:
				log.Println("Unsupported rtcp feedback")
				continue

			case webrtc.TypeRTCPFBTransportCC:
				if strings.HasPrefix(t.Codec().MimeType, "video") {
					for _, ext := range recv.GetParameters().HeaderExtensions {
						if ext.URI == sdp.TransportCCURI {
							twccExt = uint8(ext.ID)
							break
						}
					}
				}
			}
		}

		peerContext.twcc = twcc.NewTransportWideCCResponder(uint32(t.SSRC()))
		peerContext.twcc.OnFeedback(func(pkts []rtcp.Packet) {
			if err := peerContext.subscriber.peerConnection.WriteRTCP(pkts); err != nil {
				log.Printf("transport-cc ERROR | %s", err)
			}
		})

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

			if peerContext.twcc != nil && twccExt != 0 {
				if ext := pkt.GetExtension(twccExt); ext != nil {
					peerContext.twcc.Push(binary.BigEndian.Uint16(ext[0:2]), time.Now().UnixNano(), pkt.Marker)
				}
			}

			subscribers := subscriberPool.Get()

			ParallelExec(subscribers, threshold, step, func(sub *Subscriber) {
				select {
				case <-subscriberConnCtx.Done():
					return
				default:
				}

				var track TrackContextWritable
				var exist bool
				var err error

				switch {
				case subscriber.id == sub.id:
					track, exist = sub.GetLoopbackTrack(t.ID())
					if !exist {
						track, err = sub.LoopbackTrack(t.ID(), t.StreamID(), t.Codec().RTPCodecCapability)
						break
					}
					// TODO: if I will send stub track I need do this too
					// loopback.sender.ReplaceTrack(track webrtc.TrackLocal)
				default:
					track, exist = sub.HasTrack(t.ID())
					if !exist {
						track, err = sub.CreateTrack(t)
					}
				}

				if err == nil && !exist {
					// 	// TODO: need try do it by https://github.com/pion/webrtc/blob/v3.2.24/peerconnection.go#L290
					go signalPeerConnections()
				}

				if track == nil {
					return
				}

				// WriteRTP takes about 50µs
				track.WriteRTP(pkt)
			})
		}
	})

	peerContext.subscriber.peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateFailed:
			subscriberConnCtxCancel(ErrOnStateClosed)
			if err := peerContext.subscriber.peerConnection.Close(); err != nil {
				log.Print(err)
			}
		case webrtc.PeerConnectionStateClosed:
			subscriberConnCtxCancel(ErrOnStateClosed)
			signalPeerConnections()
		}
	})

	listLock.Lock()
	state := &peerConnectionState{
		peerContext:    peerContext,
		peerConnection: peerConnection,
		signaling:      &webrtc.DataChannel{},
		ws:             w,
	}
	peerConnections[peerContext.subscriber.id] = state
	defer delete(peerConnections, peerContext.subscriber.id)
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
		case "candidate":
			{
				candidate := webrtc.ICECandidateInit{}
				if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
					log.Println("Wrong ice candidate format", err)
					return err
				}
				if err := subscriber.peerConnection.AddICECandidate(candidate); err != nil {
					log.Println("Uanble add ice candidate", err)
					return err
				}
			}
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
	peerConnections = make(map[string]*peerConnectionState)
	trackLocals     = make(map[string]*webrtc.TrackLocalStaticRTP)
)

type peerConnectionState struct {
	peerContext    *PeerContext
	peerConnection *webrtc.PeerConnection
	signaling      *webrtc.DataChannel
	ws             *threadSafeWriter
}

func syncAttempt() (tryAgain bool) {
	for _, conn := range peerConnections {
		log.Printf("sync attempt %+v", conn.peerContext.subscriber)

		if conn.peerContext.subscriber.peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
			continue
		}

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

func signalPeerConnections() {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		dispatchKeyFrame()
	}()

	signalPeerConnectionDebounce := debounce(syncAttempt, time.Millisecond*4)

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
			go func() {
				// time.Sleep(time.Second * 3)
				time.Sleep(time.Millisecond * 20)
				signalPeerConnections()
			}()
			return
		}

		if !signalPeerConnectionDebounce() {
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

	// RoomService protocol.RoomService
	Logger     *slog.Logger
	API        *webrtc.API
	RtpStatsCh chan *rtpstats.RtpStats
}

func NewWhipController(params newWhipController_Params) *whipController {
	return &whipController{
		// roomService: params.RoomService,
		logger: params.Logger,
		webrtc: params.API,
		stats:  params.RtpStatsCh,
	}
}
