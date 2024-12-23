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

// RoomNotifierResponse defines model for RoomNotifierResponse.
type RoomNotifierResponse = map[string]interface{}

// RoomControllerRoomCreateJSONRequestBody defines body for RoomControllerRoomCreate for application/json ContentType.
type RoomControllerRoomCreateJSONRequestBody = RoomCreateRequest

// ServerInterface represents all server handlers.
type ServerInterface interface {

	// (GET /rooms)
	RoomControllerRoomList(ctx echo.Context) error

	// (POST /rooms)
	RoomControllerRoomCreate(ctx echo.Context) error

	// (GET /rooms-notifier)
	RoomControllerRoomNotifier(ctx echo.Context) error

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

// RoomControllerRoomNotifier converts echo context to params.
func (w *ServerInterfaceWrapper) RoomControllerRoomNotifier(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.RoomControllerRoomNotifier(ctx)
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
	router.GET(baseURL+"/rooms-notifier", wrapper.RoomControllerRoomNotifier)
	router.GET(baseURL+"/rooms/:room_id", wrapper.RoomControllerRoomJoin)
	router.DELETE(baseURL+"/rooms/:sessionID", wrapper.RoomControllerRoomDelete)

}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/8xVTW/bMAz9KwO3oxdnzc23dr1kG7agOwbGoNpMoiKWVIoZFgT674Mk26mdj7rdAvQS",
	"2zFNvvf4SO2g0JXRChVbyHZgixVWItzOBLEspBGK/aMhbZBYYngpS//LW4OQgWWSagnOJUD4uJGEJWRz",
	"H5MnTYy+f8CCwSVwp3V1mNDsq8UCjFW4+UC4gAzep3ugaY0yfQrRtaUEkdj6Z9K6mg4AWsclXRCnsH8m",
	"FIx3+LhBe0SZSvyZ9bgsNFWCIQOpeHIFbV6pGJdIz0E9i8IarSwewqBa5XPyhU4cE+Mk91tcY7fq0bAv",
	"Wqpng75Jy+fxDzdCZNJ3wBFmp9v6XbNcSKQzuF0CFosNSd7+9IUj0hsUhHS94VU7QP6j+/D3vtkrZgPO",
	"55BqoUN2yWv/JrRTKya9XiO9u55NIYHfSFZqBRmMR+PRJ49SG1TCSMhgMhqPJsGvvAoY0lauJQZPeikF",
	"S628qXoVGvHByxPJhk+vxmN/KbRijDMvjFnLIuRJH6xH02yIIQ3pNDgwL9EWJA1HYj++RnuLpfXt6YKE",
	"3CVgtB3EJg4DxHaj5Rtdbv8rle7Mu66zmDboLqxlb9xPq9kYFLJ515rz3OVnxHZJbaKPqh6EF7ipmZ1L",
	"O+pgRl/hqpZouvOXX7J0L2DqN1s8KESFjGSDztIX9sMICSgRxr/ODX2jJE/o9ld9fmH5Omv536SzaP16",
	"mt4G8cpwLAzRLx4ggxRsS7wpDXsn4GtU3L/cNVx7QS53fwMAAP//ZgU0o5kJAAA=",
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
