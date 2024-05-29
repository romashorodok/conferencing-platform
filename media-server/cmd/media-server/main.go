package main

import (
	"context"
	"log"
	"net/http"

	"github.com/romashorodok/conferencing-platform/media-server/internal/identity"
	"github.com/romashorodok/conferencing-platform/media-server/internal/room"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/service"
	globalprotocol "github.com/romashorodok/conferencing-platform/pkg/protocol"
	"go.uber.org/fx"

	_ "net/http/pprof"
)

type CreateTestRoom_Params struct {
	fx.In

	*room.RoomService
}

func CreateTestRoom(params CreateTestRoom_Params) {
	log.Println("Creating test room")
	roomID := "test"
	room, err := params.RoomService.CreateRoom(&room.RoomCreateOption{
		MaxParticipants: 0,
		RoomID:          &roomID,
	})
	if err != nil {
		log.Println(err)
	}
	log.Println(room)
	_ = room
}

type CreateAdminUser_Params struct {
	fx.In

	IdentityService *identity.IdentityService
}

func CreateAdminUser(params CreateAdminUser_Params) {
	_, _ = params.IdentityService.SignUp(context.Background(), "test", "test")
}

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	fx.New(
		fx.Provide(

			room.NewRoomService,
			room.NewRoomNotifier,

			identity.NewTokenService,
			identity.NewIdentityService,

			globalprotocol.AsHttpController(room.NewRoomController),
			globalprotocol.AsHttpController(identity.NewIdentityController),
		),

		fx.Module("test-room",
			fx.Invoke(CreateTestRoom),
			fx.Invoke(CreateAdminUser),
		),

		service.LoggerModule,
		service.DatabaseModule,
		service.WebrtcModule,
		service.HttpModule,
	).Run()
}
