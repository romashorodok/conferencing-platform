package protocol

import (
	echo "github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

const httpControllerTag = `group:"http.controller"`

type HttpRouter = *echo.Echo

// Help resolve http handler. It's needed for providing router into handler
type HttpResolvable interface {
	Resolve(HttpRouter) error
}

func AsHttpController(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(HttpResolvable)),
		fx.ResultTags(httpControllerTag),
	)
}
