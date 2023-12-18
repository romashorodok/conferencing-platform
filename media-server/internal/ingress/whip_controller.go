package ingress

import (
	echo "github.com/labstack/echo/v4"
	"github.com/romashorodok/conferencing-platform/pkg/controller/ingress"
	"github.com/romashorodok/conferencing-platform/pkg/protocol"
	"go.uber.org/fx"
)

type whipController struct{}

func (*whipController) WebrtcHttpIngestionControllerWebrtcHttpIngest(ctx echo.Context, sessionID string) error {
	panic("unimplemented")
}

func (*whipController) WebrtcHttpIngestionControllerWebrtcHttpTerminate(ctx echo.Context, sessionID string) error {
	panic("unimplemented")
}

func (ctrl *whipController) Resolve(c *echo.Echo) error {
	spec, err := ingress.GetSwagger()
	if err != nil {
		return err
	}
	spec.Servers = nil
	ingress.RegisterHandlers(c, ctrl)
	return nil
}

var (
	_ ingress.ServerInterface = (*whipController)(nil)
	_ protocol.HttpResolvable = (*whipController)(nil)
)

type newWhipController_Params struct {
	fx.In
}

func NewWhipController(params newWhipController_Params) *whipController {
	return &whipController{}
}
