package roomcontext

import (
	"context"

	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/pkg/controller/room"
)

type roomContext struct {
	roomID protocol.RoomID

	ctx    context.Context
	cancel context.CancelCauseFunc
}

func (r *roomContext) Info() *room.Room {
	return &room.Room{
		Host:         new(string),
		Participants: &[]room.Participant{},
		Port:         new(string),
		SessionID:    &r.roomID,
	}
}

func (r *roomContext) Cancel(reason error) {
	r.cancel(reason)
}

var _ protocol.RoomContext = (*roomContext)(nil)

type RoomContextOption struct {
	RoomID     protocol.RoomID
	RoomOption *protocol.RoomCreateOption
}

func NewRoomContext(roomOption RoomContextOption) *roomContext {
	ctx, cancel := context.WithCancelCause(context.TODO())

	return &roomContext{
		roomID: roomOption.RoomID,
		ctx:    ctx,
		cancel: cancel,
	}
}
