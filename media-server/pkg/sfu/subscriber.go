package sfu

import (
	"context"
	"errors"
	"sync"

	webrtc "github.com/pion/webrtc/v3"
)

type Subscriber struct {
	webrtc         *webrtc.API
	peerConnection *webrtc.PeerConnection
	peerId         string

	sid string

	// loopback map[string]*LoopbackTrackContext
	tracks           map[string]*TrackContext
	tracksMu         sync.Mutex
	pipeAllocContext *AllocatorsContext

	ctx    context.Context
	cancel context.CancelCauseFunc
}

// Create track context with default RTP sender
func (s *Subscriber) Track(t *webrtc.TrackRemote, recv *webrtc.RTPReceiver, filter *Filter) (*TrackContext, error) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	id := t.ID()

	if track, exist := s.tracks[id]; exist {
		if track != nil {
			return track, nil
		}
	}

	// NOTE: Track may have same id, but it may have different layerID(RID)
	// trackRtp, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, id, t.StreamID())
	// if err != nil {
	// 	log.Println("unable create track for rtp.", err)
	// 	return nil, err
	// }
	//
	// trackSample, err := webrtc.NewTrackLocalStaticSample(t.Codec().RTPCodecCapability, id, t.StreamID())
	// if err != nil {
	// 	log.Println("unable create track sample.")
	// 	return nil, err
	// }

	// var sender *webrtc.RTPSender
	// err = nil
	// switch filter {
	// case FILTER_NONE:
	// 	sender, err = s.peerConnection.AddTrack(trackRtp)
	// default:
	// 	sender, err = s.peerConnection.AddTrack(trackSample)
	// }
	// if err != nil {
	// 	log.Println("unable add track to the subscriber", err)
	// 	return nil, err
	// }

	// var twccExt uint8
	// for _, fb := range t.Codec().RTCPFeedback {
	// 	switch fb.Type {
	// 	case webrtc.TypeRTCPFBGoogREMB:
	// 	case webrtc.TypeRTCPFBNACK:
	// 		log.Printf("Unsupported rtcp feedbacak %s type", fb.Type)
	// 		continue
	//
	// 	case webrtc.TypeRTCPFBTransportCC:
	// 		if strings.HasPrefix(t.Codec().MimeType, "video") {
	// 			for _, ext := range recv.GetParameters().HeaderExtensions {
	// 				if ext.URI == sdp.TransportCCURI {
	// 					twccExt = uint8(ext.ID)
	// 					break
	// 				}
	// 			}
	// 		}
	// 	}
	// }

	trackContext := NewTrackContext(s.ctx, NewTrackContextParams{
		ID:          id,
		StreamID:    t.StreamID(),
		RID:         t.RID(),
		SSRC:        t.SSRC(),
		PayloadType: t.PayloadType(),

		CodecParams: t.Codec(),
		Kind:        t.Kind(),

		// TrackRtp:         trackRtp,
		// TrackSample:      trackSample,

		Filter:           filter,
		PipeAllocContext: s.pipeAllocContext,

		PeerConnection: s.peerConnection,
		API:            s.webrtc,

		// Track:  track,
		// Sender: sender,
		// TWCC_EXT: twccExt,
		// SSRC:     uint32(t.SSRC()),
	})
	s.tracks[t.ID()] = trackContext
	return trackContext, nil
}

func (s *Subscriber) Attach(t *TrackContext) (*TrackContext, error) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	if track, exist := s.tracks[t.ID()]; exist {
		if track != nil {
			return track, nil
		}
	}

	s.tracks[t.ID()] = t
	return t, nil
}

// func (s *Subscriber) LoopbackTrack(t *webrtc.TrackRemote, recv *webrtc.RTPReceiver) (*LoopbackTrackContext, error) {
// 	trackCtx, err := s.Track(t, recv)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	s.loopback[trackCtx.track.ID()] = &LoopbackTrackContext{
// 		TrackContext: trackCtx,
// 	}
//
// 	return s.loopback[trackCtx.track.ID()], nil
// }

func (s *Subscriber) HasTrack(trackID string) (*TrackContext, bool) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	track, exist := s.tracks[trackID]
	return track, exist
}

// func (s *Subscriber) HasLoopbackTrack(trackID string) (track *LoopbackTrackContext, exist bool) {
// 	s.tracksMu.Lock()
// 	defer s.tracksMu.Unlock()
//
// 	track, exist = s.loopback[trackID]
// 	return
// }

func (s *Subscriber) DeleteTrack(trackID string) (err error) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	track, exist := s.tracks[trackID]
	if !exist {
		return errors.New("Track not exist. Unable delete")
	}
	err = track.Close()
	delete(s.tracks, trackID)
	return err
}

func (s *Subscriber) Close() (err error) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	for id, t := range s.tracks {
		err = s.peerConnection.RemoveTrack(t.sender)
		err = t.Close()
		delete(s.tracks, id)
	}

	// for id, t := range s.loopback {
	// 	err = s.peerConnection.RemoveTrack(t.sender)
	// 	err = t.Close()
	// 	delete(s.loopback, id)
	// }

	return err
}
