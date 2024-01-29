package protocol

type RoomID = string

// type (
// 	RoomInfo     = room.Room
// 	RoomInfoList = []room.Room
// )

// RoomContext represents the room state and provides access to mutate it.
// type RoomContext struct{}

type RoomCreateOption struct {
	MaxParticipants int32
	RoomID          *string
}

// type RoomService interface {
// 	GetRoom(RoomID) RoomContext
// 	CreateRoom(*RoomCreateOption) (RoomContext, error)
// 	DeleteRoom(RoomID) error
// 	ListRoom() []RoomInfoList
// }
