package room

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/roomcontext"
	"github.com/romashorodok/conferencing-platform/pkg/controller/room"
	"go.uber.org/fx"
)

var (
	RoomNotExistError     error = errors.New("room not exist")
	RoomCancelByUserError error = errors.New("room canceled by user")
)

type roomService struct {
	sync.RWMutex
	logger *slog.Logger

	roomContextMap map[protocol.RoomID]protocol.RoomContext
}

// ListRoom implements protocol.RoomService.
func (s *roomService) ListRoom() []room.Room {
	var result []room.Room
	for _, room := range s.roomContextMap {
		result = append(result, *room.Info())
	}
	return result
}

func (s *roomService) DeleteRoom(roomID string) error {
	room, exist := s.roomContextMap[roomID]
	if !exist {
		return RoomNotExistError
	}
	room.Cancel(RoomCancelByUserError)
	return nil
}

func (s *roomService) CreateRoom(option *protocol.RoomCreateOption) (protocol.RoomContext, error) {
	s.Lock()
	defer s.Unlock()

	roomID := uuid.New().String()
	s.roomContextMap[roomID] = roomcontext.NewRoomContext(
		roomcontext.RoomContextOption{
			RoomID:     roomID,
			RoomOption: option,
		},
	)
	room, exist := s.roomContextMap[roomID]
	if !exist && room == nil {
		return nil, errors.New("not found room or it's nil")
	}
	return room, nil
}

var _ protocol.RoomService = (*roomService)(nil)

type NewRoomService_Params struct {
	fx.In

	Logger *slog.Logger
}

func NewRoomService(params NewRoomService_Params) *roomService {
	return &roomService{
		logger:         params.Logger,
		roomContextMap: make(map[protocol.RoomID]protocol.RoomContext),
	}
}
