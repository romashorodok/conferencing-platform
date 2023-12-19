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

	RoomService protocol.RoomService
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

func main() {
	fx.New(
		fx.Provide(
			globalprotocol.AsHttpController(ingress.NewWhipController),
			globalprotocol.AsHttpController(room.NewRoomController),

			fx.Annotate(
				room.NewRoomService,
				fx.As(new(protocol.RoomService)),
			),
		),

		fx.Module("test-room",
			fx.Invoke(CreateTestRoom),
		),

		service.LoggerModule,
		service.WebrtcModule,
		service.HttpModule,
	).Run()
}
