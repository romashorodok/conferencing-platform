package main

import (
	"log"

	"github.com/romashorodok/conferencing-platform/media-server/internal/ingress"
	"github.com/romashorodok/conferencing-platform/media-server/internal/room"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/service"
	globalprotocol "github.com/romashorodok/conferencing-platform/pkg/protocol"
	"go.uber.org/fx"
)

type CreateTestRoom_Params struct {
	fx.In

	*room.RoomService
}

func CreateTestRoom(params CreateTestRoom_Params) {
	log.Println("Creating test room")
	roomID := "test"
	room, err := params.RoomService.CreateRoom(&protocol.RoomCreateOption{
		MaxParticipants: 0,
		RoomID:          &roomID,
	})
	if err != nil {
		log.Println(err)
	}
	log.Println(room)
	_ = room
}

type roomStateChangedChanResult struct {
	fx.Out

	RoomStateChanged chan struct{} `name:"room_state_changed"`
}

func roomStateChangedChan() roomStateChangedChanResult {
	return roomStateChangedChanResult{
		RoomStateChanged: make(chan struct{}),
	}
}

func main() {
	fx.New(
		fx.Provide(
			roomStateChangedChan,
			room.NewRoomService,

			globalprotocol.AsHttpController(ingress.NewWhipController),
			globalprotocol.AsHttpController(room.NewRoomController),
		),

		fx.Module("test-room",
			fx.Invoke(CreateTestRoom),
		),

		service.LoggerModule,
		service.WebrtcModule,
		service.HttpModule,
	).Run()
}
