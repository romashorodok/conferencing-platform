package ingress

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	echo "github.com/labstack/echo/v4"
	"github.com/pion/rtcp"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/pkg/controller/ingress"
	globalprotocol "github.com/romashorodok/conferencing-platform/pkg/protocol"
	"go.uber.org/fx"
)

var INGEST_ANSWER_TYPE = "answer"

type whipController struct {
	roomService protocol.RoomService
	logger      *slog.Logger
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
	publisher           *webrtc.PeerConnection
	subscriber          *webrtc.PeerConnection
	negotiationDataChan *webrtc.DataChannel
}

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
			_ = peerContext.subscriber.Close()
		}
	}()

	subscriber, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Println("Unable create subscriber peer")
		return err
	}
	peerContext.subscriber = subscriber
	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := subscriber.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Print(err)
			return err
		}
	}

	// Need send ice candidate to the client for success gathering
	peerContext.subscriber.OnICECandidate(func(i *webrtc.ICECandidate) {
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
			Event: "trickle-ice-candidate",
			Data:  string(candidateString),
		})
	})

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
		case "publish":
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

				if err := peerContext.subscriber.SetRemoteDescription(webrtc.SessionDescription{
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
				dc, err := subscriber.CreateDataChannel("_negotiation", nil)
				if err != nil {
					log.Println(err)
					return err
				}
				peerContext.negotiationDataChan = dc

				offer, err := subscriber.CreateOffer(nil)
				if err != nil {
					log.Println(err)
					return err
				}
				if err = subscriber.SetLocalDescription(offer); err != nil {
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
	peerConnection *webrtc.PeerConnection
	signaling      *webrtc.DataChannel
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

// signalPeerConnections updates each PeerConnection so that it is getting all the expected media tracks
func signalPeerConnections() {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		dispatchKeyFrame()
	}()

	attemptSync := func() (tryAgain bool) {
		for i := range peerConnections {
			if peerConnections[i].peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				peerConnections = append(peerConnections[:i], peerConnections[i+1:]...)
				return true // We modified the slice, start from the beginning
			}

			// map of sender we already are seanding, so we don't double send
			existingSenders := map[string]bool{}

			for _, sender := range peerConnections[i].peerConnection.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				// If we have a RTPSender that doesn't map to a existing track remove and signal
				if _, ok := trackLocals[sender.Track().ID()]; !ok {
					if err := peerConnections[i].peerConnection.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}
			log.Println(peerConnections[i].peerConnection.GetReceivers())

			// Don't receive videos we are sending, make sure we don't have loopback
			for _, receiver := range peerConnections[i].peerConnection.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}

				existingSenders[receiver.Track().ID()] = true
			}

			// Add all track we aren't sending yet to the PeerConnection
			for trackID := range trackLocals {
				if _, ok := existingSenders[trackID]; !ok {
					if _, err := peerConnections[i].peerConnection.AddTrack(trackLocals[trackID]); err != nil {
						return true
					}
				}
			}
			log.Println(existingSenders)

			offer, err := peerConnections[i].peerConnection.CreateOffer(nil)
			if err != nil {
				return true
			}

			<-webrtc.GatheringCompletePromise(peerConnections[i].peerConnection)
			// you can't break the flow of a new RTCPeerConnections (create offer -> set local -> set remote -> create answer -> set local -> set remote). If you create offer on one side, answer on the other and only then set the local/remote descriptions it should break by design.

			if err = peerConnections[i].peerConnection.SetLocalDescription(offer); err != nil {
				return true
			}

			offerString, err := json.Marshal(offer)
			if err != nil {
				return true
			}
			if err = peerConnections[i].signaling.SendText(string(offerString)); err != nil {
				return true
			}
		}

		return
	}

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
			go func() {
				time.Sleep(time.Second * 3)
				signalPeerConnections()
			}()
			log.Println("Failed sync")
			return
		}

		if !attemptSync() {
			log.Println("success sync")
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

func (ctrl *whipController) WebrtcHttpIngestionControllerWebrtcHttpIngest(ctx echo.Context, sessionID string) error {
	var request ingress.WebrtcHttpIngestRequest

	if err := json.NewDecoder(ctx.Request().Body).Decode(&request); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Print("peerConnection", err)
		return err
	}

	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionSendrecv,
		}); err != nil {
			log.Print("transivers errors", err)
			return err
		}
	}

	peerConnectionDataChannel := &peerConnectionState{
		peerConnection: peerConnection,
		signaling:      nil,
	}

	listLock.Lock()
	peerConnections = append(peerConnections, peerConnectionDataChannel)
	listLock.Unlock()

	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		peerConnectionDataChannel.signaling = dc

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			type localPeerAnswer struct {
				Type string
				Sdp  string
			}

			var answer localPeerAnswer
			json.Unmarshal(msg.Data, &answer)

			desc := webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  answer.Sdp,
			}

			if err := peerConnection.SetLocalDescription(desc); err != nil {
				log.Println("Unable set answer", err)
				return
			}
			log.Println("Success answer set")
		})
	})

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		trackLocal := addTrack(track)
		defer removeTrack(trackLocal)

		buf := make([]byte, 1500)
		for {
			i, _, err := track.Read(buf)
			if err != nil {
				return
			}

			if _, err = trackLocal.Write(buf[:i]); err != nil {
				return
			}
		}
	})

	err = peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  *request.Offer.Sdp,
	})
	if err != nil {
		log.Println("Set remote desc", err)
		return err
	}

	answer, err := peerConnection.CreateAnswer(nil)

	peerConnection.SetLocalDescription(answer)

	<-webrtc.GatheringCompletePromise(peerConnection)

	ctx.JSON(http.StatusCreated,
		&ingress.WebrtcHttpIngestResponse{
			Answer: &ingress.SessionDescription{
				Sdp:  &peerConnection.LocalDescription().SDP,
				Type: &INGEST_ANSWER_TYPE,
			},
		},
	)

	return nil
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

	RoomService protocol.RoomService
	Logger      *slog.Logger
}

func NewWhipController(params newWhipController_Params) *whipController {
	return &whipController{
		roomService: params.RoomService,
		logger:      params.Logger,
	}
}
