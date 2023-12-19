package service

import (
	"fmt"
	"log/slog"

	echo "github.com/labstack/echo/v4"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/variables"
	"github.com/romashorodok/conferencing-platform/pkg/protocol"
	"go.uber.org/fx"
)

type httpServer_Params struct {
	fx.In

	Controllers []protocol.HttpResolvable `group:"http.controller"`
	Logger      *slog.Logger
}

func httpErrorHandler(e *echo.Echo, logger *slog.Logger) func(err error, c echo.Context) {
	return func(err error, c echo.Context) {
		logger.Error(err.Error(), slog.String("request", fmt.Sprintf("%+v", c.Request())))
		e.DefaultHTTPErrorHandler(err, c)
	}
}

func httpServer(params httpServer_Params) {
	router := echo.New()
	router.HTTPErrorHandler = httpErrorHandler(router, params.Logger)

	for _, controller := range params.Controllers {
		controller.Resolve(router)
	}

	router.Logger.Fatal(router.Start(fmt.Sprintf(":%s", variables.Env(variables.HTTP_PORT_NAME, variables.HTTP_PORT_DEFAULT))))
}

var HttpModule = fx.Module("http", fx.Invoke(httpServer))
