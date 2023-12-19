package roomcontext

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	webrtc "github.com/pion/webrtc/v4"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/peercontext"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/pkg/controller/room"
)

type roomContext struct {
	sync.Mutex

	roomID protocol.RoomID

	peerContextMap map[protocol.PeerID]protocol.PeerContext

	logger    *slog.Logger
	webrtcAPI *webrtc.API
	ctx       context.Context
	cancel    context.CancelCauseFunc
}

func (r *roomContext) AddParticipant(offer string) (protocol.PeerContext, error) {
	r.Lock()
	defer r.Unlock()

	peerID := uuid.New().String()
	peerContext, err := peercontext.NewPeerContext(
		peercontext.NewPeerContext_Params{
			API:           r.webrtcAPI,
			Logger:        r.logger,
			ParentContext: r.ctx,
			PeerID:        peerID,
		},
	)
	if err != nil {
		return nil, err
	}

	if err := peerContext.SetRemoteSessionDescriptor(offer); err != nil {
		peerContext.Cancel(err)
		return nil, err
	}
	peerContext.OnDataChannel()

	r.peerContextMap[peerID] = peerContext

	return peerContext, nil
}

func (r *roomContext) Info() *room.Room {
	var participants []room.Participant

	for _, peerContext := range r.peerContextMap {
		info := peerContext.Info()

		participants = append(participants, room.Participant{
			Id:   &info.ID,
			Name: &info.Name,
		})
	}

	return &room.Room{
		Host:         new(string),
		Participants: &participants,
		Port:         new(string),
		SessionID:    &r.roomID,
	}
}

func (r *roomContext) Cancel(reason error) {
	r.cancel(reason)
}

var _ protocol.RoomContext = (*roomContext)(nil)

type RoomContextOption struct {
	WebrtcAPI  *webrtc.API
	Logger     *slog.Logger
	RoomID     protocol.RoomID
	RoomOption *protocol.RoomCreateOption
}

func NewRoomContext(roomOption RoomContextOption) *roomContext {
	ctx, cancel := context.WithCancelCause(context.TODO())

	return &roomContext{
		roomID:         roomOption.RoomID,
		peerContextMap: make(map[string]protocol.PeerContext),
		logger:         roomOption.Logger,
		webrtcAPI:      roomOption.WebrtcAPI,
		ctx:            ctx,
		cancel:         cancel,
	}
}
