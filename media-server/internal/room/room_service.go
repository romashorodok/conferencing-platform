package room

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/sfu"
	"github.com/romashorodok/conferencing-platform/pkg/controller/room"
	"github.com/romashorodok/conferencing-platform/pkg/executils"
	"github.com/romashorodok/conferencing-platform/pkg/wsutils"
	"go.uber.org/fx"
)

var (
	ErrRoomAlreadyExists = errors.New("room already exists")
	ErrRoomNotExist      = errors.New("room not exist")
	ErrRoomCancelByUser  = errors.New("room canceled by user")
)

type RoomNotifier struct {
	listeners     map[string]*wsutils.ThreadSafeWriter
	updateRoomCh  chan struct{}
	updateRoomsMu sync.Mutex
}

func (n *RoomNotifier) Listen(id string, w *wsutils.ThreadSafeWriter) {
	n.updateRoomsMu.Lock()
	defer n.updateRoomsMu.Unlock()
	n.listeners[id] = w
}

func (n *RoomNotifier) Stop(id string) {
	delete(n.listeners, id)
}

func (n *RoomNotifier) DispatchUpdateRooms() {
	n.updateRoomsMu.Lock()
	defer n.updateRoomsMu.Unlock()

	if len(n.listeners) == 0 {
		return
	}

	n.updateRoomCh <- struct{}{}
}

func (n *RoomNotifier) getListeners() (result []*wsutils.ThreadSafeWriter) {
	for _, listener := range n.listeners {
		result = append(result, listener)
	}
	return
}

func (n *RoomNotifier) OnUpdateRooms(ctx context.Context, fn func(*wsutils.ThreadSafeWriter)) {
	var threshold uint64 = 1000000
	var step uint64 = 2
	for {
		select {
		case <-ctx.Done():
			return
		case <-n.updateRoomCh:
			executils.ParallelExec(n.getListeners(), threshold, step, fn)
		}
	}
}

func NewRoomNotifier() *RoomNotifier {
	return &RoomNotifier{
		listeners:    make(map[string]*wsutils.ThreadSafeWriter),
		updateRoomCh: make(chan struct{}),
	}
}

type roomContext struct {
	roomID          protocol.RoomID
	peerContextPool *sfu.PeerContextPool
}

func (r *roomContext) Info() room.Room {
	participants := make([]room.Participant, 0)

	for _, p := range r.peerContextPool.Get() {
		participants = append(participants, room.Participant{
			Id: p.PeerID,
		})
	}

	return room.Room{
		RoomId:       r.roomID,
		Participants: participants,
	}
}

type NewRoomContextParams struct {
	RoomID protocol.RoomID
}

func NewRoomContext(params NewRoomContextParams) *roomContext {
	return &roomContext{
		roomID:          params.RoomID,
		peerContextPool: sfu.NewPeerContextPool(),
	}
}

type RoomService struct {
	sync.Mutex

	webrtcAPI      *webrtc.API
	logger         *slog.Logger
	roomContextMap map[protocol.RoomID]*roomContext
	roomNotifier   *RoomNotifier
}

func (s *RoomService) GetRoom(roomID string) *roomContext {
	room, exist := s.roomContextMap[roomID]
	if !exist {
		return nil
	}
	return room
}

func (s *RoomService) ListRoom() []room.Room {
	result := make([]room.Room, 0)
	for _, room := range s.roomContextMap {
		result = append(result, room.Info())
	}
	return result
}

//
// func (s *roomService) DeleteRoom(roomID string) error {
// 	room, exist := s.roomContextMap[roomID]
// 	if !exist {
// 		return ErrRoomNotExist
// 	}
// 	room.Cancel(ErrRoomCancelByUser)
// 	delete(s.roomContextMap, roomID)
// 	return nil
// }

func NullableRoomID(roomID *string) string {
	if roomID != nil && *roomID != "" {
		return *roomID
	}
	return uuid.NewString()
}

func (s *RoomService) CreateRoom(option *protocol.RoomCreateOption) (*roomContext, error) {
	s.Lock()
	defer s.Unlock()

	roomID := NullableRoomID(option.RoomID)
	if _, exist := s.roomContextMap[roomID]; exist {
		return nil, ErrRoomAlreadyExists
	}

	s.roomContextMap[roomID] = NewRoomContext(NewRoomContextParams{
		RoomID: roomID,
	})

	room, exist := s.roomContextMap[roomID]
	if !exist && room == nil {
		return nil, errors.New("not found room or it's nil")
	}

	s.roomNotifier.DispatchUpdateRooms()

	return room, nil
}

type NewRoomServiceParams struct {
	fx.In

	WebrtcAPI    *webrtc.API
	Logger       *slog.Logger
	RoomNotifier *RoomNotifier
}

func NewRoomService(params NewRoomServiceParams) *RoomService {
	return &RoomService{
		webrtcAPI:      params.WebrtcAPI,
		logger:         params.Logger,
		roomContextMap: make(map[string]*roomContext),
		roomNotifier:   params.RoomNotifier,
	}
}
