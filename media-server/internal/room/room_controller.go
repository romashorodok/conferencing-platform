package room

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	echo "github.com/labstack/echo/v4"
	webrtc "github.com/pion/webrtc/v4"
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

type filterData struct {
	Enabled  bool   `json:"enabled"`
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
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
	cookies := ctx.Request().Cookies()
	log.Printf("cookies %+v", cookies)

	roomCtx := ctrl.roomService.GetRoom(roomId)
	if roomCtx == nil {
		return ErrRoomNotExist
	}

	conn, err := ctrl.upgrader.Upgrade(ctx.Response().Writer, ctx.Request(), nil)
	if err != nil {
		ctrl.logger.Error(fmt.Sprintf("Unable upgrade request %+v", ctx.Request()))
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
		Spreader:         roomCtx.peerContextPool,
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

	peerContext.OnTrack()
	peerContext.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		candidate, lErr := json.Marshal(c.ToJSON())
		if lErr != nil {
			log.Println(lErr)
			return
		}

		if err := w.WriteJSON(websocketMessage{
			Event: "candidate",
			Data:  string(candidate),
		}); err != nil {
			ctrl.wsError(w, err)
			return
		}
	})

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
			roomCtx.peerContextPool.Remove(peerContext)
			roomCtx.peerContextPool.DispatchOffers()
		}
	})

	go func() {
		ticker := time.NewTicker(time.Second * 10)
		for {
			select {
			case <-peerContext.Done():
				return
			case <-ticker.C:
				roomCtx.peerContextPool.SanitizePeerSenders(peerContext)
				log.Println("dispatch offer")
			}
		}
	}()

	if _, err := peerContext.CreateDataChannel("_negotiation", nil); err != nil {
		return ctrl.wsError(w, err)
	}

	go func() {
	retry:
		peerFilters := peerContext.Filters()

		filtersBytes, err := json.Marshal(peerFilters)
		if err != nil {
			time.Sleep(time.Second)
			goto retry
		}

		if err = w.WriteJSON(&websocketMessage{
			Event: "filters",
			Data:  string(filtersBytes),
		}); err != nil {
			select {
			case <-peerContext.Done():
				return
			default:
				time.Sleep(time.Second)
				goto retry
			}
		}
	}()

	go peerContext.SynchronizeOfferState()

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

		case "commit-offer-state":
			var offerState sfu.CommitOfferStateMessage
			if err := json.Unmarshal([]byte(message.Data), &offerState); err != nil {
				return ctrl.wsError(w, err)
			}

			log.Println("Offer State recv,", offerState.StateHash)
			if err := peerContext.CommitOfferState(offerState); err != nil {
				log.Println("[commit-offer-state] Commit offer state. Err:", err)
				// return ctrl.wsError(w, err)
			}

		case "filter":
			var fData filterData
			if err := json.Unmarshal([]byte(message.Data), &fData); err != nil {
				return ctrl.wsError(w, err)
			}

			if err := peerContext.SwitchFilter(fData.Name, fData.MimeType); err != nil {
				log.Println(err)
				return ctrl.wsError(w, err)
			}

		default:
			return ctrl.wsError(w, errors.New("wrong message event"))
		}
	}
}

func (*roomController) RoomControllerRoomDelete(ctx echo.Context, sessionID string) error {
	panic("unimplemented")
}

func (ctrl *roomController) RoomControllerRoomList(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, room.RoomListResponse{
		Rooms: ctrl.roomService.ListRoom(),
	})
}

type RoomCreateOption struct {
	MaxParticipants int32
	RoomID          *string
}

func (ctrl *roomController) RoomControllerRoomCreate(ctx echo.Context) error {
	var request room.RoomCreateRequest
	if err := json.NewDecoder(ctx.Request().Body).Decode(&request); err != nil {
		return err
	}

	room, err := ctrl.roomService.CreateRoom(&RoomCreateOption{
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
