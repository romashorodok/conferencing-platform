package main

import (
	"log"

	"github.com/romashorodok/conferencing-platform/media-server/cmd/media-server/cpppipelines"
	"github.com/romashorodok/conferencing-platform/media-server/internal/ingress"
	"github.com/romashorodok/conferencing-platform/media-server/internal/room"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/service"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/sfu"
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

func NewPipelinesAllocatorsContext() *sfu.AllocatorsContext {
	allocContext := sfu.NewAllocatorsContext()
	allocContext.Register(sfu.FILTER_RTP_VP8_DUMMY, sfu.Allocator(cpppipelines.NewRtpVP8))
	return allocContext
}

func main() {
	cpppipelines.GstreamerMainLoopSetup()

	fx.New(
		fx.Provide(
			NewPipelinesAllocatorsContext,

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
