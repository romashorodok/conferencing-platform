package identity

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	globalprotocol "github.com/romashorodok/conferencing-platform/pkg/protocol"
	"go.uber.org/fx"
)

type errResponse struct {
	Message string `json:"message"`
}

func newErrorResponse(err error) any {
	return errResponse{
		Message: err.Error(),
	}
}

type identityController struct {
	identityService *IdentityService
}

type identitySignInRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (i *identityController) IdentitySignIn(c echo.Context) error {
	req := new(identitySignInRequest)
	if err := c.Bind(req); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}

	tokenPair, err := i.identityService.SignIn(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		log.Println("SignIn", tokenPair, "err", err)

		switch {
		case errors.Is(err, sql.ErrNoRows):
			return c.JSON(http.StatusNotFound, &errResponse{
				Message: "Invalid user credentials",
			})
		default:
		}
		return c.JSON(http.StatusInternalServerError, &errResponse{
			Message: err.Error(),
		})
	}

	refreshTokenCookieExpires := time.Now().AddDate(1, 0, 0)
	refreshTokenCookie := http.Cookie{
		Name:     REFRESH_TOKEN_HTTP_ONLY_COOKIE_NAME,
		Value:    tokenPair.refreshToken,
		Expires:  refreshTokenCookieExpires,
		Secure:   true,
		HttpOnly: true,
		Path:     "*",
	}
	c.SetCookie(&refreshTokenCookie)

	return c.JSON(http.StatusOK, tokenPair)
}

func (i *identityController) IdentitySignOut(echo.Context) error {
	panic("unimplemented")
}

type identitySignUpRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (i *identityController) IdentitySignUp(c echo.Context) error {
	req := new(identitySignUpRequest)
	if err := c.Bind(req); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}

	tokenPair, err := i.identityService.SignUp(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		log.Println("SignUp", tokenPair, "err", err)
		return err
	}

	refreshTokenCookieExpires := time.Now().AddDate(1, 0, 0)
	refreshTokenCookie := &http.Cookie{
		Name:     REFRESH_TOKEN_HTTP_ONLY_COOKIE_NAME,
		Value:    tokenPair.refreshToken,
		Expires:  refreshTokenCookieExpires,
		Secure:   true,
		HttpOnly: true,
		Path:     "*",
	}
	c.SetCookie(refreshTokenCookie)

	return c.JSON(http.StatusOK, tokenPair)
}

type identityTokenVerifyRequestHeader struct {
	Authorization string `header:"authorization"`
}

type identityTokenVerifyValidResponse struct {
	Verified bool `json:"verified"`
}

var _ErrEmptyAuthorizationHeader = errors.New("empty authorization header")

func (i *identityController) IdentityTokenVerify(c echo.Context) error {
	token := WithTokenContext(c)

	if token.TokenUse != REFRESH_TOKEN {
		return c.JSON(http.StatusOK, &identityTokenVerifyValidResponse{
			Verified: true,
		})
	}

	tokenPair, err := i.identityService.ActualizeTokenPair(c.Request().Context(), token)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, newErrorResponse(err))
	}

	refreshTokenCookieExpires := time.Now().AddDate(1, 0, 0)
	refreshTokenCookie := &http.Cookie{
		Name:     REFRESH_TOKEN_HTTP_ONLY_COOKIE_NAME,
		Value:    tokenPair.refreshToken,
		Expires:  refreshTokenCookieExpires,
		Secure:   true,
		HttpOnly: true,
		Path:     "*",
	}
	c.SetCookie(refreshTokenCookie)

	return c.JSON(http.StatusCreated, tokenPair)
}

func (i *identityController) wallEcho(c echo.Context) error {
	return c.JSON(http.StatusOK, WithTokenContext(c))
}

type identityWrapper interface {
	IdentitySignIn(echo.Context) error
	IdentitySignUp(c echo.Context) error
	IdentitySignOut(echo.Context) error
	IdentityTokenVerify(echo.Context) error
}

func (i *identityController) Resolve(router *echo.Echo) error {
	baseURL := "/identity"

	middlewares := []echo.MiddlewareFunc{
		echo.MiddlewareFunc(IdentityWallFactoryMiddleware(i.identityService)),
	}

	httpOnlyMiddleware := []echo.MiddlewareFunc{
		echo.MiddlewareFunc(IdentityHttpOnlyCookieWallFactoryMiddleware(i.identityService)),
	}

	router.POST(baseURL+"/sign-in", i.IdentitySignIn)
	router.POST(baseURL+"/sign-up", i.IdentitySignUp)
	router.DELETE(baseURL+"/sign-out", i.IdentitySignOut, middlewares...)

	router.POST(baseURL+"/token-verify", i.IdentityTokenVerify, httpOnlyMiddleware...)
	router.POST(baseURL+"/wall-echo", i.wallEcho, middlewares...)

	return nil
}

var (
	_ globalprotocol.HttpResolvable = (*identityController)(nil)
	_ identityWrapper               = (*identityController)(nil)
)

type newIdentityControllerParams struct {
	fx.In

	IdentityService *IdentityService
}

func NewIdentityController(params newIdentityControllerParams) *identityController {
	return &identityController{
		identityService: params.IdentityService,
	}
}
