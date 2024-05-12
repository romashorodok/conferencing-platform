package identity

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type identityWallMiddlewareHeaders struct {
	Authorization string `header:"authorization"`
}

var echoDefaultBinder = &echo.DefaultBinder{}

type MiddlewareFactory func(echo.HandlerFunc) echo.HandlerFunc

type identityResolver interface {
	TokenIdentity(ctx context.Context, insecureToken string) (*TokenContext, error)
}

const _TOKEN_CONTEXT_KEY = "TOKEN_CONTEXT"

func WithTokenContext(c echo.Context) *TokenContext {
	return c.Get(_TOKEN_CONTEXT_KEY).(*TokenContext)
}

func IdentityWallFactoryMiddleware(resolver identityResolver) MiddlewareFactory {
	//
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			headers := new(identityWallMiddlewareHeaders)

			if err := echoDefaultBinder.BindHeaders(c, headers); err != nil {
				return c.JSON(http.StatusInternalServerError, &errResponse{
					Message: fmt.Sprintf("Unable bind headers to pass identity wall. Err: %s", err),
				})
			}

			insecureToken := strings.TrimPrefix(headers.Authorization, "Bearer ")

			if headers.Authorization == "" || headers.Authorization == insecureToken {
				return c.JSON(http.StatusPreconditionFailed, &errResponse{
					Message: "Missing authorization header",
				})
			}

			token, err := resolver.TokenIdentity(c.Request().Context(), insecureToken)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, &errResponse{
					Message: fmt.Sprintf("Identity resolving failed. Err: %s", err),
				})
			}

			c.Set(_TOKEN_CONTEXT_KEY, token)

			return next(c)
		}
	}
}
