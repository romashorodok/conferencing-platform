package protocol

import (
	"github.com/romashorodok/conferencing-platform/pkg/controller/room"
)

type RoomID = string

type (
	RoomInfo     = *room.Room
	RoomInfoList = room.Room
)

// RoomContext represents the room state and provides access to mutate it.
type RoomContext interface {
	Cancel(error)
	Info() RoomInfo
}

type RoomCreateOption struct {
	MaxParticipants int32
}

type RoomService interface {
	CreateRoom(*RoomCreateOption) (RoomContext, error)
	DeleteRoom(RoomID) error
	ListRoom() []RoomInfoList
}
