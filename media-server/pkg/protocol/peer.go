package protocol

import (
	"errors"

	webrtc "github.com/pion/webrtc/v3"
)

type PeerID = string

type PeerInfo struct {
	ID   string
	Name string
}

var ErrPeerSignalingChannelNotFound = errors.New("peer don't have signaling channel")

var PeerSignalingChannelLabel = "signaling"

type PeerContext interface {
	SetRemoteSessionDescriptor(offer string) error
	GenerateSDPAnswer() (string, error)
	// TODO: refactor SFU
	GetPeerConnection() *webrtc.PeerConnection
	GetSignalingChannel() (*webrtc.DataChannel, error)

	OnDataChannel()
	OnCandidate()
	OnTrack(signaling func())
	Info() *PeerInfo
}
