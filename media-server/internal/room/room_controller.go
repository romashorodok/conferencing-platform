package room

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

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	echo "github.com/labstack/echo/v4"
	"github.com/pion/rtcp"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/rtpstats"
	"github.com/romashorodok/conferencing-platform/pkg/controller/room"
	globalprotocol "github.com/romashorodok/conferencing-platform/pkg/protocol"
	"go.uber.org/atomic"
	"go.uber.org/fx"
)

var (
	ErrOnStateClosed           = errors.New("Closed connection")
	ErrUnsupportedMessageEvent = errors.New("Unsupported message event")
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

type SdpAnswer struct {
	Type string `json:"type"`
	Sdp  string `json:"sdp"`
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

func (t *threadSafeWriter) Close() error {
	return t.Conn.Close()
}

func (t *threadSafeWriter) ReadJSON(val any) error {
	return t.Conn.ReadJSON(val)
}

func (ctrl *roomController) wsError(w *threadSafeWriter, err error) error {
	ctrl.logger.Error(fmt.Sprintf("%s | Err: %s", w.Conn.RemoteAddr(), err))
	w.WriteJSON(&websocketMessage{
		Event: "error",
		Data:  "wrong data format",
	})
	return err
}

type roomController struct {
	lifecycle        fx.Lifecycle
	roomService      *RoomService
	stats            <-chan *rtpstats.RtpStats
	upgrader         websocket.Upgrader
	webrtc           *webrtc.API
	logger           *slog.Logger
	peerConnectionMu sync.Mutex
	roomNotifier     *RoomNotifier
}

func (ctrl *roomController) RoomControllerRoomNotifier(ctx echo.Context) error {
	conn, err := ctrl.upgrader.Upgrade(ctx.Response().Writer, ctx.Request(), nil)
	if err != nil {
		ctrl.logger.Error(fmt.Sprintf("Unable upgrade request %s", ctx.Request()))
		return err
	}

	w := &threadSafeWriter{Conn: conn}
	defer w.Close()

	id := uuid.NewString()
	ctrl.roomNotifier.Listen(id, w)
	defer ctrl.roomNotifier.Stop(id)

	for {
		select {
		case <-ctx.Request().Context().Done():
			return ErrRoomCancelByUser
		}
	}
}

func (ctrl *roomController) RoomControllerRoomJoin(ctx echo.Context, roomId string) error {
	roomCtx := ctrl.roomService.GetRoom(roomId)
	if roomCtx == nil {
		return ErrRoomNotExist
	}

	conn, err := ctrl.upgrader.Upgrade(ctx.Response().Writer, ctx.Request(), nil)
	if err != nil {
		ctrl.logger.Error(fmt.Sprintf("Unable upgrade request %s", ctx.Request()))
		return err
	}

	w := &threadSafeWriter{Conn: conn}
	defer w.Close()

	peerContext := NewPeerContext(NewPeerContextParams{
		Parent: ctx.Request().Context(),
		API:    ctrl.webrtc,
		WS:     w,
	})
	ctrl.peerConnectionMu.Lock()
	if err := peerContext.NewPeerConnection(); err != nil {
		return ctrl.wsError(w, err)
	}
	defer peerContext.Cancel(errors.New("Implicit connection close"))
	defer peerContext.Close()
	defer func() {
		roomCtx.peerContextPool.Remove(peerContext)
		ctrl.roomNotifier.DispatchUpdateRooms()
	}()
	peerContext.stats = <-ctrl.stats
	ctrl.peerConnectionMu.Unlock()
	peerContext.NewSubscriber()

	// Declare publisher transceiver for all kinds
	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peerContext.peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			return ctrl.wsError(w, err)
		}
	}

	// Need send ice candidate to the client for success gathering
	peerContext.peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
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

	peerContext.peerConnection.OnTrack(func(t *webrtc.TrackRemote, recv *webrtc.RTPReceiver) {
		defer func() {
			for _, ctx := range roomCtx.peerContextPool.Get() {
				_ = ctx.Subscriber.DeleteTrack(t.ID())
			}
		}()

		var threshold uint64 = 1000000
		var step uint64 = 2
		log.Println("On track", t.ID())

		onTrackFeedback := func(pkts []rtcp.Packet) {
			if err := peerContext.peerConnection.WriteRTCP(pkts); err != nil {
				log.Printf("transport-cc ERROR | %s", err)
			}
		}

		for {
			select {
			case <-peerContext.Ctx.Done():
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

			peerContexts := roomCtx.peerContextPool.Get()

			ParallelExec(peerContexts, threshold, step, func(peer *PeerContext) {
				select {
				case <-peer.Ctx.Done():
					return
				default:
				}

				var track TrackCongestionControlWritable
				var exist bool
				var err error

				switch {
				case peerContext.peerID == peer.peerID:
					track, exist = peer.Subscriber.GetLoopbackTrack(t.ID())
					if !exist {
						log.Println("Create loopback track")
						track, err = peer.Subscriber.LoopbackTrack(t, recv)
						track.OnFeedback(onTrackFeedback)
					}

				default:
					track, exist = peer.Subscriber.HasTrack(t.ID())
					if !exist {
						log.Println("Create local track track")
						track, err = peer.Subscriber.CreateTrack(t, recv)
						track.OnFeedback(onTrackFeedback)
					}
				}

				if err == nil && !exist {
					go peer.SignalPeerConnection()
				}

				if track == nil {
					return
				}

				// WriteRTP takes about 50µs
				track.WriteRTP(pkt)
			})
		}
	})

	peerContext.peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateConnected:
			if err = roomCtx.peerContextPool.Add(peerContext); err != nil {
				ctrl.wsError(w, err)
				peerContext.Cancel(errors.New("Unable add into pool. On PeerConnectionStateConnected"))
				return
			}
			ctrl.roomNotifier.DispatchUpdateRooms()

		case webrtc.PeerConnectionStateFailed:
			peerContext.Close()
			peerContext.Cancel(ErrOnStateClosed)
		case webrtc.PeerConnectionStateClosed:
			peerContext.Close()
			peerContext.Cancel(ErrOnStateClosed)
			roomCtx.peerContextPool.SignalPeerContexts()
		}
	})

	message := &websocketMessage{}
	for {
		if err := w.ReadJSON(message); err != nil {
			return ctrl.wsError(w, err)
		}
		select {
		case <-peerContext.Ctx.Done():
			return peerContext.Ctx.Err()
		default:
		}

		switch message.Event {
		case "candidate":
			candidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
				return ctrl.wsError(w, err)
			}

			if err := peerContext.peerConnection.AddICECandidate(candidate); err != nil {
				return ctrl.wsError(w, err)
			}

		case "answer": /* Get answer from subscriber client side now peer already connected if they now ICE */

			var answer SdpAnswer
			if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
				return ctrl.wsError(w, err)
			}

			if err := peerContext.peerConnection.SetRemoteDescription(webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  answer.Sdp,
			}); err != nil {
				return ctrl.wsError(w, err)
			}

		case "subscribe": /* Send initial offer when someone start subscribing */

			if _, err := peerContext.peerConnection.CreateDataChannel("_negotiation", nil); err != nil {
				return ctrl.wsError(w, err)
			}

			offer, err := peerContext.peerConnection.CreateOffer(nil)
			if err != nil {
				return ctrl.wsError(w, err)
			}

			if err = peerContext.peerConnection.SetLocalDescription(offer); err != nil {
				return ctrl.wsError(w, err)
			}

			offerJSON, err := json.Marshal(offer)
			if err != nil {
				return ctrl.wsError(w, err)
			}

			if err = w.WriteJSON(&websocketMessage{
				Event: "offer",
				Data:  string(offerJSON),
			}); err != nil {
				return ctrl.wsError(w, err)
			}
		}
	}
}

// RoomControllerRoomDelete implements room.ServerInterface.
func (*roomController) RoomControllerRoomDelete(ctx echo.Context, sessionID string) error {
	panic("unimplemented")
}

func (ctrl *roomController) RoomControllerRoomList(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, room.RoomListResponse{
		Rooms: ctrl.roomService.ListRoom(),
	})
}

func (ctrl *roomController) RoomControllerRoomCreate(ctx echo.Context) error {
	var request room.RoomCreateRequest
	if err := json.NewDecoder(ctx.Request().Body).Decode(&request); err != nil {
		return err
	}

	room, err := ctrl.roomService.CreateRoom(&protocol.RoomCreateOption{
		RoomID:          request.RoomId,
		MaxParticipants: *request.MaxParticipants,
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, room.Info())
}

// func (ctrl *roomController) RoomControllerRoomDelete(ctx echo.Context, sessionID string) error {
// 	err := ctrl.roomService.DeleteRoom(sessionID)
// 	if err != nil {
// 		log.Println(err)
// 		return err
// 	}
// 	ctx.JSON(http.StatusCreated, make(room.RoomDeleteResponse))
// 	return nil
// }

// func (ctrl *roomController) RoomControllerRoomList(ctx echo.Context) error {
// 	result := ctrl.roomService.ListRoom()
// 	ctx.JSON(http.StatusOK, &room.RoomListResponse{
// 		Rooms: &result,
// 	})
// 	return nil
// }

func (ctrl *roomController) Resolve(c *echo.Echo) error {
	go ctrl.roomNotifier.OnUpdateRooms(context.Background(), func(w *threadSafeWriter) {
		w.WriteJSON(&websocketMessage{
			Event: "update-rooms",
			Data:  "",
		})
	})

	spec, err := room.GetSwagger()
	if err != nil {
		return err
	}
	spec.Servers = nil
	room.RegisterHandlers(c, ctrl)
	return nil
}

var (
	_ room.ServerInterface          = (*roomController)(nil)
	_ globalprotocol.HttpResolvable = (*roomController)(nil)
)

type newRoomController_Params struct {
	fx.In
	Lifecycle fx.Lifecycle

	RoomService  *RoomService
	API          *webrtc.API
	Logger       *slog.Logger
	Stats        chan *rtpstats.RtpStats
	RoomNotifier *RoomNotifier
}

func NewRoomController(params newRoomController_Params) *roomController {
	return &roomController{
		lifecycle:   params.Lifecycle,
		webrtc:      params.API,
		stats:       params.Stats,
		logger:      params.Logger,
		roomService: params.RoomService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		roomNotifier: params.RoomNotifier,
	}
}
