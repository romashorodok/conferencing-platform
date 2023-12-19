package room

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	webrtc "github.com/pion/webrtc/v4"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/roomcontext"
	"github.com/romashorodok/conferencing-platform/pkg/controller/room"
	"go.uber.org/fx"
)

var (
	ErrRoomNotExist     error = errors.New("room not exist")
	ErrRoomIDIsEmpty    error = errors.New("room id is empty")
	ErrRoomCancelByUser error = errors.New("room canceled by user")
)

type roomService struct {
	sync.Mutex

	webrtcAPI      *webrtc.API
	logger         *slog.Logger
	roomContextMap map[protocol.RoomID]protocol.RoomContext
}

func (s *roomService) GetRoom(roomID string) protocol.RoomContext {
	room, exist := s.roomContextMap[roomID]
	if !exist {
		return nil
	}
	return room
}

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
		return ErrRoomNotExist
	}
	room.Cancel(ErrRoomCancelByUser)
	delete(s.roomContextMap, roomID)
	return nil
}

func (s *roomService) CreateRoom(option *protocol.RoomCreateOption) (protocol.RoomContext, error) {
	s.Lock()
	defer s.Unlock()

	var roomID string
	if option.RoomID != nil {
		if *option.RoomID == "" {
			return nil, ErrRoomIDIsEmpty
		}
		roomID = *option.RoomID
	} else {
		roomID = uuid.New().String()
	}

	s.roomContextMap[roomID] = roomcontext.NewRoomContext(
		roomcontext.RoomContextOption{
			WebrtcAPI:  s.webrtcAPI,
			Logger:     s.logger,
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

	WebrtcAPI *webrtc.API
	Logger    *slog.Logger
}

func NewRoomService(params NewRoomService_Params) *roomService {
	return &roomService{
		webrtcAPI:      params.WebrtcAPI,
		logger:         params.Logger,
		roomContextMap: make(map[protocol.RoomID]protocol.RoomContext),
	}
}
