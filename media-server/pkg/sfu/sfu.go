package sfu

import (
	"sync"

	webrtc "github.com/pion/webrtc/v3"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
)

type SelectiveForwardingUnit struct {
	sync.Mutex

	TrackLocalList map[protocol.PeerID]*webrtc.TrackLocalStaticRTP
}

func (sfu *SelectiveForwardingUnit) AddTrack(remoteTrack *webrtc.TrackRemote) (*webrtc.TrackLocalStaticRTP, error) {
	sfu.Lock()
	defer sfu.Unlock()

	localTrack, err := webrtc.NewTrackLocalStaticRTP(
		remoteTrack.Codec().RTPCodecCapability,
		remoteTrack.ID(),
		remoteTrack.StreamID(),
	)
	if err != nil {
		return nil, err
	}

	sfu.TrackLocalList[localTrack.ID()] = localTrack

	return localTrack, nil
}

func (sfu *SelectiveForwardingUnit) RemoveTrack(track *webrtc.TrackLocalStaticRTP) {
	sfu.Lock()
	defer sfu.Unlock()
	delete(sfu.TrackLocalList, track.ID())
}

func NewSelectiveForwardUnit() *SelectiveForwardingUnit {
	return &SelectiveForwardingUnit{
		TrackLocalList: make(map[string]*webrtc.TrackLocalStaticRTP),
	}
}
