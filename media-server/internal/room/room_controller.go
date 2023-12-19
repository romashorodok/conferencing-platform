package room

import (
	"log"
	"net/http"

	echo "github.com/labstack/echo/v4"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/pkg/controller/room"
	globalprotocol "github.com/romashorodok/conferencing-platform/pkg/protocol"
	"go.uber.org/fx"
)

type roomController struct {
	roomService protocol.RoomService
}

func (c *roomController) RoomControllerRoomCreate(ctx echo.Context) error {
	c.roomService.CreateRoom(&protocol.RoomCreateOption{
		MaxParticipants: 0,
	})
	return nil
}

func (ctrl *roomController) RoomControllerRoomDelete(ctx echo.Context, sessionID string) error {
	err := ctrl.roomService.DeleteRoom(sessionID)
	if err != nil {
		log.Println(err)
		return err
	}
	ctx.JSON(http.StatusCreated, make(room.RoomDeleteResponse))
	return nil
}

func (ctrl *roomController) RoomControllerRoomList(ctx echo.Context) error {
	result := ctrl.roomService.ListRoom()
	ctx.JSON(http.StatusOK, &room.RoomListResponse{
		Rooms: &result,
	})
	return nil
}

func (ctrl *roomController) Resolve(c *echo.Echo) error {
	spec, err := room.GetSwagger()
	if err != nil {
		return err
	}
	spec.Servers = nil
	room.RegisterHandlers(c, ctrl)
	return nil
}

var (
	_ room.ServerInterface          = (*roomController)(nil)
	_ globalprotocol.HttpResolvable = (*roomController)(nil)
)

type newRoomController_Params struct {
	fx.In

	RoomService protocol.RoomService
}

func NewRoomController(params newRoomController_Params) *roomController {
	return &roomController{
		roomService: params.RoomService,
	}
}
