// Package room provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.16.2 DO NOT EDIT.
package room

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime"
)

const (
	BearerAuthScopes = "BearerAuth.Scopes"
)

// Participant defines model for Participant.
type Participant struct {
	Id string `json:"id"`
}

// Room defines model for Room.
type Room struct {
	Participants []Participant `json:"participants"`
	RoomId       string        `json:"roomId"`
}

// RoomCreateRequest defines model for RoomCreateRequest.
type RoomCreateRequest struct {
	MaxParticipants *int32  `json:"maxParticipants,omitempty"`
	RoomId          *string `json:"roomId,omitempty"`
}

// RoomCreateResponse defines model for RoomCreateResponse.
type RoomCreateResponse struct {
	Room Room `json:"room"`
}

// RoomDeleteResponse defines model for RoomDeleteResponse.
type RoomDeleteResponse = map[string]interface{}

// RoomJoinResponse defines model for RoomJoinResponse.
type RoomJoinResponse = map[string]interface{}

// RoomListResponse defines model for RoomListResponse.
type RoomListResponse struct {
	Rooms []Room `json:"rooms"`
}

// RoomControllerRoomCreateJSONRequestBody defines body for RoomControllerRoomCreate for application/json ContentType.
type RoomControllerRoomCreateJSONRequestBody = RoomCreateRequest

// ServerInterface represents all server handlers.
type ServerInterface interface {

	// (GET /rooms)
	RoomControllerRoomList(ctx echo.Context) error

	// (POST /rooms)
	RoomControllerRoomCreate(ctx echo.Context) error

	// (GET /rooms/{room_id})
	RoomControllerRoomJoin(ctx echo.Context, roomId string) error

	// (DELETE /rooms/{sessionID})
	RoomControllerRoomDelete(ctx echo.Context, sessionID string) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// RoomControllerRoomList converts echo context to params.
func (w *ServerInterfaceWrapper) RoomControllerRoomList(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.RoomControllerRoomList(ctx)
	return err
}

// RoomControllerRoomCreate converts echo context to params.
func (w *ServerInterfaceWrapper) RoomControllerRoomCreate(ctx echo.Context) error {
	var err error

	ctx.Set(BearerAuthScopes, []string{})

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.RoomControllerRoomCreate(ctx)
	return err
}

// RoomControllerRoomJoin converts echo context to params.
func (w *ServerInterfaceWrapper) RoomControllerRoomJoin(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "room_id" -------------
	var roomId string

	err = runtime.BindStyledParameterWithLocation("simple", false, "room_id", runtime.ParamLocationPath, ctx.Param("room_id"), &roomId)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter room_id: %s", err))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.RoomControllerRoomJoin(ctx, roomId)
	return err
}

// RoomControllerRoomDelete converts echo context to params.
func (w *ServerInterfaceWrapper) RoomControllerRoomDelete(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "sessionID" -------------
	var sessionID string

	err = runtime.BindStyledParameterWithLocation("simple", false, "sessionID", runtime.ParamLocationPath, ctx.Param("sessionID"), &sessionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter sessionID: %s", err))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.RoomControllerRoomDelete(ctx, sessionID)
	return err
}

// This is a simple interface which specifies echo.Route addition functions which
// are present on both echo.Echo and echo.Group, since we want to allow using
// either of them for path registration
type EchoRouter interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router EchoRouter, si ServerInterface) {
	RegisterHandlersWithBaseURL(router, si, "")
}

// Registers handlers, and prepends BaseURL to the paths, so that the paths
// can be served under a prefix.
func RegisterHandlersWithBaseURL(router EchoRouter, si ServerInterface, baseURL string) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.GET(baseURL+"/rooms", wrapper.RoomControllerRoomList)
	router.POST(baseURL+"/rooms", wrapper.RoomControllerRoomCreate)
	router.GET(baseURL+"/rooms/:room_id", wrapper.RoomControllerRoomJoin)
	router.DELETE(baseURL+"/rooms/:sessionID", wrapper.RoomControllerRoomDelete)

}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/8yUT2/bPAzGv8oLvjsacdbcfGvXS7YBC7pjYAyqzSQqYkmlmGFBoO8+UMo/u07qFSuw",
	"S2zHMvnwx4fcQWUbZw0a9lDswFcrbFS8nSliXWmnDMujI+uQWGN8qWv55a1DKMAzabOEEDIgfN5owhqK",
	"uZwps8MZ+/iEFUPI4MHa5mVAd8qWEjA28eYD4QIK+D8/Cc33KvNzieGYShGprTyTtc10gND9uawt4pL2",
	"T4SK8QGfN+h7yDTq16xTy8JSoxgK0IYnN3CMqw3jEuk1qVdVeGeNx5cyaE/5Gr7YiT4YF2u/xzW2s/Ye",
	"+2y1efXQV+35uv7hRkiVdB3QU1lfW0MGHqsNad5+l3hJwB0qQrrd8Oo4F/LRY/z71MMVs4MgMbRZ2Fis",
	"5rW8iV2yhsmu10j/3c6mkMFPJK+tgQLGo/Hoo2i2Do1yGgqYjMajSbQhr6KG/EhhidFqQkixtka80slw",
	"YApSdcIaP70Zj+VSWcOYRlk5t9ZVjJM/eVFzGPwhnFt9i5XX6CvSjlNh374k16qlF+ptkVCGDJz1g6pJ",
	"HofURfR8Z+vtXy2lPcqhbRimDYZ3ZtmZ4ss0DwaFYt625rwM5RXYIdubKN/J5Yeuwx/YSeY4rUXVICP5",
	"mF6LLvEoZGBUnIp9bOjyy85YdBdb+c5sW0voDT49Q+fRy9RO7yO8Oi7BIfzSuhxE8Jjin2LY2fdvoXh6",
	"uTvU2jkUyvA7AAD//5kb0tSHCAAA",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
