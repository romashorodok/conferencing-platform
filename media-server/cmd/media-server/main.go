package main

import (
	"context"
	"log"
	"net/http"

	"github.com/romashorodok/conferencing-platform/media-server/internal/identity"
	"github.com/romashorodok/conferencing-platform/media-server/internal/ingress"
	"github.com/romashorodok/conferencing-platform/media-server/internal/mcu"
	"github.com/romashorodok/conferencing-platform/media-server/internal/pipeline"
	"github.com/romashorodok/conferencing-platform/media-server/internal/room"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/service"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/sfu"
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

type CreateAdminUser_Params struct {
	fx.In

	IdentityService *identity.IdentityService
}

func CreateAdminUser(params CreateAdminUser_Params) {
	_, _ = params.IdentityService.SignUp(context.Background(), "admin", "admin")
}

var _ sfu.Pipeline = (*pipeline.CannyFilter)(nil)

func NewPipelinesAllocatorsContext() *sfu.AllocatorsContext {
	allocContext := sfu.NewAllocatorsContext()
	allocContext.Register(sfu.FILTER_RTP_CANNY_FILTER, pipeline.NewCannyFilter)
	return allocContext
}

func main() {
	mcu.Setup()
	mcu.Version()
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	fx.New(
		fx.Provide(
			NewPipelinesAllocatorsContext,

			room.NewRoomService,
			room.NewRoomNotifier,

			identity.NewTokenService,
			identity.NewIdentityService,

			globalprotocol.AsHttpController(ingress.NewWhipController),
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
