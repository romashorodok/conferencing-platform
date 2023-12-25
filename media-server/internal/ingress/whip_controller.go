package ingress

import (
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"net/http"

	echo "github.com/labstack/echo/v4"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/pkg/controller/ingress"
	globalprotocol "github.com/romashorodok/conferencing-platform/pkg/protocol"
	"go.uber.org/fx"
)

var INGEST_ANSWER_TYPE = "answer"

type whipController struct {
	roomService protocol.RoomService
	logger      *slog.Logger
}

const (
	MISSING_ICE_UFRAG_MSG = "offer must contain at least one track or data channel"
	NOT_FOUND_ROOM_MSG    = "not found room"
)

func (ctrl *whipController) WebrtcHttpIngestionControllerWebrtcHttpIngest(ctx echo.Context, sessionID string) error {
	var request ingress.WebrtcHttpIngestRequest

	if err := json.NewDecoder(ctx.Request().Body).Decode(&request); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	room := ctrl.roomService.GetRoom(sessionID)
	if room == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, errors.New(NOT_FOUND_ROOM_MSG))
	}

	peer, err := room.AddParticipant(*request.Offer.Sdp)
	if err != nil {
		if errors.Is(err, webrtc.ErrSessionDescriptionMissingIceUfrag) {
			return echo.NewHTTPError(http.StatusPreconditionFailed, MISSING_ICE_UFRAG_MSG)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peer.GetPeerConnection().AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			// Direction: webrtc.RTPTransceiverDirectionSendonly,
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Print(err)
			return err
		}
	}

	answer, err := peer.GenerateSDPAnswer()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	ctx.JSON(http.StatusCreated,
		&ingress.WebrtcHttpIngestResponse{
			Answer: &ingress.SessionDescription{
				Sdp:  &answer,
				Type: &INGEST_ANSWER_TYPE,
			},
		},
	)

	return nil
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
	_ ingress.ServerInterface       = (*whipController)(nil)
	_ globalprotocol.HttpResolvable = (*whipController)(nil)
)

type newWhipController_Params struct {
	fx.In

	RoomService protocol.RoomService
	Logger      *slog.Logger
}

func NewWhipController(params newWhipController_Params) *whipController {
	return &whipController{
		roomService: params.RoomService,
		logger:      params.Logger,
	}
}
