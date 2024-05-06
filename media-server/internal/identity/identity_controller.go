package identity

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	globalprotocol "github.com/romashorodok/conferencing-platform/pkg/protocol"
	"go.uber.org/fx"
)

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
	log.Println("SignIn", tokenPair, "err", err)
	if err != nil {
		return err
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
	log.Println("SignUp", tokenPair, "err", err)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, tokenPair)
}

func (i *identityController) IdentityTokenVerify(echo.Context) error {
	panic("unimplemented")
}

type identityWrapper interface {
	IdentitySignIn(echo.Context) error
	IdentitySignOut(echo.Context) error
	IdentityTokenVerify(echo.Context) error
}

func (i *identityController) Resolve(router *echo.Echo) error {
	baseURL := "/identity"

	router.POST(baseURL+"/sign-in", i.IdentitySignIn)
	router.POST(baseURL+"/sign-up", i.IdentitySignUp)
	router.DELETE(baseURL+"/sign-out", i.IdentitySignOut)
	router.POST(baseURL+"/token-verify", i.IdentityTokenVerify)

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
