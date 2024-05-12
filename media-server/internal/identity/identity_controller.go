package identity

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

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

	pair, err := i.identityService.ActualizeTokenPair(c.Request().Context(), token)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, newErrorResponse(err))
	}

	return c.JSON(http.StatusCreated, pair)
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

	router.POST(baseURL+"/sign-in", i.IdentitySignIn)
	router.POST(baseURL+"/sign-up", i.IdentitySignUp)
	router.DELETE(baseURL+"/sign-out", i.IdentitySignOut)

	router.POST(baseURL+"/token-verify", i.IdentityTokenVerify, middlewares...)
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
