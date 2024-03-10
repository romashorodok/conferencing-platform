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

/*
#cgo LDFLAGS: -lstdc++

#cgo pkg-config: pipelines-1.0
#cgo pkg-config: rtpvp8-1.0
#include "pipelines.h"
#include "rtpvp8/rtpvp8.h"
*/
import "C"

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

func main() {
	C.setup()
	C.print_version()
	fx.New(
		fx.Provide(
			room.NewRoomService,
			room.NewRoomNotifier,

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
