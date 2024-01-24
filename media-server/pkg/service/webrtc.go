package service

import (
	"log/slog"

	ice "github.com/pion/ice/v3"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/variables"
	"go.uber.org/fx"
)

type webrtcAPI_Params struct {
	fx.In
}

var (
	ONE_TO_NAT_PUBLIC_IP = variables.Env(
		variables.WEBRTC_ONE_TO_NAT_PUBLIC_IP,
		variables.WEBRTC_ONE_TO_NAT_PUBLIC_IP_DEFAULT,
	)

	WEBRTC_PORT = variables.Env(
		variables.WEBRTC_UDP_PORT,
		variables.WEBRTC_UDP_PORT_DEFAULT,
	)
)

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

	udpPort, err := variables.ParseInt(WEBRTC_PORT)
	if err != nil {
		return nil, err
	}

	udpMux, err := ice.NewMultiUDPMuxFromPort(udpPort)
	if err != nil {
		return nil, err
	}

	mediaSettings.SetICEUDPMux(udpMux)

	if ONE_TO_NAT_PUBLIC_IP != "" {
		mediaSettings.SetNAT1To1IPs([]string{ONE_TO_NAT_PUBLIC_IP}, webrtc.ICECandidateTypeHost)
	}

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
