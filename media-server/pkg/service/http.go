package service

import (
	echo "github.com/labstack/echo/v4"
	"github.com/romashorodok/conferencing-platform/pkg/protocol"
	"go.uber.org/fx"
)

type httpServer_Params struct {
	fx.In

	Controllers []protocol.HttpResolvable `group:"http.controller"`
}

func httpServer(params httpServer_Params) {
	router := echo.New()

	for _, controller := range params.Controllers {
		controller.Resolve(router)
	}

	router.Logger.Fatal(router.Start(":4200"))
}

var HttpModule = fx.Module("http", fx.Invoke(httpServer))
