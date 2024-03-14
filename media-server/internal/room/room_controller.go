package room

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	echo "github.com/labstack/echo/v4"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/rtpstats"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/sfu"
	"github.com/romashorodok/conferencing-platform/pkg/controller/room"
	globalprotocol "github.com/romashorodok/conferencing-platform/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/pkg/wsutils"
	"go.uber.org/fx"
)

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

func (ctrl *roomController) wsError(w *wsutils.ThreadSafeWriter, err error) error {
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
	pipeAllocContext *sfu.AllocatorsContext
}

func (ctrl *roomController) RoomControllerRoomNotifier(ctx echo.Context) error {
	conn, err := ctrl.upgrader.Upgrade(ctx.Response().Writer, ctx.Request(), nil)
	if err != nil {
		ctrl.logger.Error(fmt.Sprintf("Unable upgrade request %s", ctx.Request()))
		return err
	}

	w := wsutils.NewThreadSafeWriter(conn)
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

type SubscribeMessage struct {
	RestartICE bool `json:"restartICE"`
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

	w := wsutils.NewThreadSafeWriter(conn)
	defer w.Close()

	ctrl.peerConnectionMu.Lock()
	peerContext, err := sfu.NewPeerContext(sfu.NewPeerContextParams{
		Context:          ctx.Request().Context(),
		API:              ctrl.webrtc,
		WS:               w,
		PipeAllocContext: ctrl.pipeAllocContext,
	})
	peerContext.SetStats(<-ctrl.stats)
	ctrl.peerConnectionMu.Unlock()
	if err != nil {
		return ctrl.wsError(w, err)
	}
	defer func() {
		peerContext.Close(sfu.ErrPeerConnectionClosed)
		roomCtx.peerContextPool.Remove(peerContext)
		ctrl.roomNotifier.DispatchUpdateRooms()
	}()

	if err = peerContext.AddTransceiver([]webrtc.RTPCodecType{
		webrtc.RTPCodecTypeVideo,
		webrtc.RTPCodecTypeAudio,
	}); err != nil {
		return ctrl.wsError(w, err)
	}

	peerContext.OnTrack(roomCtx.peerContextPool)

	peerContext.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateConnected:
			if err = roomCtx.peerContextPool.Add(peerContext); err != nil {
				ctrl.wsError(w, err)
				peerContext.Close(errors.Join(errors.New("unable add into pool."), sfu.ErrPeerConnectionClosed))
				return
			}
			ctrl.roomNotifier.DispatchUpdateRooms()

		case webrtc.PeerConnectionStateClosed, webrtc.PeerConnectionStateFailed:
			peerContext.Close(sfu.ErrPeerConnectionClosed)
			roomCtx.peerContextPool.DispatchOffers()
		}
	})

	// TODO: make on each peer track context done and wait when track is done make signal with removing the track from each peer side
	// NOTE: this fix the bug when track is removed and on client side it's on remote description
	// go func() {
	// 	ticker := time.NewTicker(time.Second * 10)
	// 	for {
	// 		select {
	// 		case <-peerContext.ctx.Done():
	// 			return
	// 		case <-ticker.C:
	// 			peerContext.SignalPeerConnection()
	// 		}
	// 	}
	// }()

	if _, err := peerContext.CreateDataChannel("_negotiation", nil); err != nil {
		return ctrl.wsError(w, err)
	}

	message := &websocketMessage{}
	for {
		if err := w.ReadJSON(message); err != nil {
			return ctrl.wsError(w, err)
		}

		select {
		case <-peerContext.Done():
			return peerContext.Err()
		default:
		}

		switch message.Event {
		case "candidate":
			if err := peerContext.Signal.OnCandidate([]byte(message.Data)); err != nil {
				return ctrl.wsError(w, err)
			}
		case "answer":
			if err := peerContext.Signal.OnAnswer([]byte(message.Data)); err != nil {
				return ctrl.wsError(w, err)
			}
		case "subscribe":
			if err := peerContext.Signal.DispatchOffer(); err != nil {
				return ctrl.wsError(w, err)
			}
		default:
			return ctrl.wsError(w, errors.New("wrong message event"))
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
	go ctrl.roomNotifier.OnUpdateRooms(context.Background(), func(w *wsutils.ThreadSafeWriter) {
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

	RoomService      *RoomService
	API              *webrtc.API
	Logger           *slog.Logger
	Stats            chan *rtpstats.RtpStats
	RoomNotifier     *RoomNotifier
	PipeAllocContext *sfu.AllocatorsContext
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
		roomNotifier:     params.RoomNotifier,
		pipeAllocContext: params.PipeAllocContext,
	}
}
