package service

import (
	"log/slog"

	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	webrtc "github.com/pion/webrtc/v3"
	"go.uber.org/fx"
)

type webrtcAPI_Params struct {
	fx.In
}

func webrtcAPI(params webrtcAPI_Params) (*webrtc.API, error) {
	mediaEngine := &webrtc.MediaEngine{}
	err := mediaEngine.RegisterDefaultCodecs()
	if err != nil {
		return nil, err
	}

	mediaSettings := webrtc.SettingEngine{}
	mediaSettings.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeUDP4,
	})

	interceptorRegistry := &interceptor.Registry{}
	pli, err := intervalpli.NewReceiverInterceptor()
	if err != nil {
		return nil, err
	}
	interceptorRegistry.Add(pli)

	if err := webrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry); err != nil {
		return nil, err
	}

	return webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(mediaSettings),
	), nil
}

var WebrtcModule = fx.Module("webrtc", fx.Provide(
	webrtcAPI,
),
	fx.Invoke(func(log *slog.Logger, api *webrtc.API) {
		log.Debug("hello world")
	}),
)
