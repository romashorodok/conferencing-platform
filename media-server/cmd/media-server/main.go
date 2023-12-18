package main

import (
	"github.com/romashorodok/conferencing-platform/media-server/internal/ingress"
	"github.com/romashorodok/conferencing-platform/media-server/internal/room"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/service"
	globalprotocol "github.com/romashorodok/conferencing-platform/pkg/protocol"
	"go.uber.org/fx"
)

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

		service.LoggerModule,
		service.WebrtcModule,
		service.HttpModule,
	).Run()
}
