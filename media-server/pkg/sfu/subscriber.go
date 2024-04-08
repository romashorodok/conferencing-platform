package sfu

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	webrtc "github.com/pion/webrtc/v4"
)

type watchTrackAck struct {
	TrackContext *TrackContext
	Result       chan error
}

func NewWatchTrackAck(t *TrackContext) watchTrackAck {
	return watchTrackAck{
		TrackContext: t,
		Result:       make(chan error),
	}
}

type Subscriber struct {
	webrtc         *webrtc.API
	peerConnection *webrtc.PeerConnection
	audioTrans     *webrtc.RTPTransceiver
	videoTrans     *webrtc.RTPTransceiver

	peerId string
	sid    string

	// tracks               map[string]*ActiveTrackContext
	tracks   sync.Map
	tracksMu sync.Mutex

	attachTrackMu        sync.Mutex
	immediateAttachTrack chan watchTrackAck
	detachTrackMu        sync.Mutex
	immediateDetachTrack chan watchTrackAck

	observers   atomic.Pointer[[]chan SubscriberMessage[any]]
	observersMu sync.Mutex

	pipeAllocContext *AllocatorsContext

	ctx    context.Context
	cancel context.CancelCauseFunc
}

func (s *Subscriber) MapStoreTrack(tID string, t *ActiveTrackContext) {
	s.tracks.Store(tID, t)
}

func (s *Subscriber) MapDeleteTrack(tID string) {
	s.tracks.Delete(tID)
}

func (s *Subscriber) MapExistTrack(tID string) (any, bool) {
	val, exist := s.tracks.Load(tID)
	return val, exist
}

func (s *Subscriber) MapForEachTrack(f func(key, value any) bool) {
	s.tracks.Range(f)
}

func (s *Subscriber) WatchTrackAttach() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case ack := <-s.immediateAttachTrack:
			s.attachTrackMu.Lock()
			t := ack.TrackContext
			// log.Println("attach track | search for", t.ID())

			track, found := s.HasTrack(t.ID())
			if !found {
				sender, err := s.peerConnection.AddTrack(t.GetLocalTrack())
				if err != nil {
					log.Printf("Ignore %s track for sub %s. Unable add track. Err:%s", t.ID(), s.peerId, err)
					ack.Result <- err
					close(ack.Result)
					s.attachTrackMu.Unlock()
					continue
				}

				// TODO: Define place where it must be
				track = NewActiveTrackContext(sender, t)
				s.MapStoreTrack(t.ID(), track)
				// s.tracks.Store(t.ID(), track)
				// s.tracks[t.ID()] = track
				ack.Result <- nil
				close(ack.Result)
				s.dispatch(NewSubscriberMessage(SubscriberTrackAttached{
					track: track,
				}))
			} else {
				log.Printf("track %s already exists", track.trackContext.ID())
				sender := track.LoadSender()

				if sender != nil {
					if err := s.peerConnection.RemoveTrack(sender); err != nil {
						log.Println(err)
						ack.Result <- err
						close(ack.Result)
						return
					}
				}

				sender, err := s.peerConnection.AddTrack(track.trackContext.GetLocalTrack())
				if err != nil {
					log.Println(err)
					ack.Result <- err
					close(ack.Result)
					return
				}

				track.StoreSender(sender)

				ack.Result <- nil
				close(ack.Result)

				s.dispatch(NewSubscriberMessage(SubscriberTrackAttached{
					track: track,
				}))
			}

			s.attachTrackMu.Unlock()
		}
	}
}

func (s *Subscriber) WatchTrackDetach() {
	for {
		select {
		case <-s.ctx.Done():
			log.Println("peer context stop detach")
			return
		case ack := <-s.immediateDetachTrack:
			s.detachTrackMu.Lock()
			t := ack.TrackContext
			// log.Println("detach track | search for", t.ID())

			track, found := s.HasTrack(t.ID())
			if !found {
				// log.Printf("detach track | not found %s track. Ignoring...", t.ID())
				ack.Result <- errors.Join(ErrWatchTrackDetachNotFound, fmt.Errorf("PeerID: %s TrackID:%s", s.peerId, t.ID()))
				close(ack.Result)
				return
			}

			t = track.trackContext
			s.MapDeleteTrack(t.ID())
			// delete(s.tracks, t.ID())

			sender := track.LoadSender()
			if sender == nil {
				// NOTE: If this called, something goes too wrong
				ack.Result <- errors.Join(ErrWatchTrackDetachSenderNotFound, fmt.Errorf("Track not attached. PeerID: %s TrackID:%s", s.peerId, t.ID()))
				close(ack.Result)
				return
			}

			err := s.peerConnection.RemoveTrack(sender)
			if err != nil {
				ack.Result <- err
			} else {
				ack.Result <- nil
			}

			s.dispatch(NewSubscriberMessage(SubscriberTrackDetached{
				track: track,
			}))

			close(ack.Result)
			s.detachTrackMu.Unlock()
		}
	}
}

// Create track context with default RTP sender
func (s *Subscriber) Track(streamID string, t *webrtc.TrackRemote, recv *webrtc.RTPReceiver, filter *Filter) watchTrackAck {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	id := t.ID()
	// NOTE: chrome may send track without id
	if id == "" {
		id = uuid.NewString()
	}

	// if track, exist := s.tracks[id]; exist {
	// 	if track != nil {
	// 		return track, nil
	// 	}
	// }

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
		ID:       id,
		StreamID: streamID,
		// StreamID:    t.StreamID(),
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
	// s.tracks[id] = trackContext
	// return trackContext, nil

	return s.AttachTrack(trackContext)
}

func (s *Subscriber) AttachTrack(t *TrackContext) watchTrackAck {
	ack := NewWatchTrackAck(t)
	s.immediateAttachTrack <- ack
	return ack
}

func (s *Subscriber) DetachTrack(t *TrackContext) watchTrackAck {
	ack := NewWatchTrackAck(t)
	s.immediateDetachTrack <- ack
	return ack
}

// func (s *Subscriber) Attach(t *TrackContext) (*TrackContext, error) {
// 	s.tracksMu.Lock()
// 	defer s.tracksMu.Unlock()
//
// 	if track, exist := s.tracks[t.ID()]; exist {
// 		if track != nil {
// 			return track, nil
// 		}
// 	}
//
// 	s.tracks[t.ID()] = t
// 	return t, nil
// }

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

var EmptyActiveTrackContext = &ActiveTrackContext{}

func (s *Subscriber) HasTrack(trackID string) (*ActiveTrackContext, bool) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	track, exist := s.MapExistTrack(trackID)
	if !exist || track == nil {
		return EmptyActiveTrackContext, exist
	}
	return track.(*ActiveTrackContext), exist
}

func (s *Subscriber) ActiveTracks() map[string]*ActiveTrackContext {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	senders := s.peerConnection.GetSenders()

	result := make(map[string]*ActiveTrackContext)

	var retryCount int
retry:
	for _, sender := range senders {
		if sender == nil {
			log.Println("[ActiveTracks] sender nil. retry", retryCount)
			goto retry
		}

		senderTrack := sender.Track()
		if senderTrack == nil {
			// if retryCount > 3 {
			// 	retryCount = 0
			// 	senders = append(senders[:idx], senders[idx+1:]...)
			//              goto retry
			// }

			log.Println("[ActiveTracks] sender track nil. retry", retryCount)
			retryCount++
			goto retry
		}

		id := senderTrack.ID()

		track, exist := s.MapExistTrack(id)
		if !exist {
			continue
		}

		result[id] = track.(*ActiveTrackContext)
	}

	return result
}

// TODO: I must delete track from peer connection not at all. Currently I have detach
// func (s *Subscriber) DeleteTrack(trackID string) (err error) {
// 	s.tracksMu.Lock()
// 	defer s.tracksMu.Unlock()
//
// 	active, exist := s.tracks[trackID]
// 	if !exist {
// 		return errors.New("Track not exist. Unable delete")
// 	}
//
// 	s.peerConnection.RemoveTrack(active.sender)
// 	err = active.trackContext.Close()
// 	delete(s.tracks, trackID)
// 	return err
// }

func (s *Subscriber) DeleteTrack(t *ActiveTrackContext) error {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	id := t.trackContext.ID()
	track, exist := s.MapExistTrack(id)
	if !exist || track == nil {
		return fmt.Errorf("Track not exist. Unable delete %s", id)
	}
	s.MapDeleteTrack(id)

	return s.peerConnection.RemoveTrack(track.(*ActiveTrackContext).LoadSender())
}

func (s *Subscriber) Close() (err error) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	s.MapForEachTrack(func(key, value any) bool {
		id := key.(string)
		active := value.(*ActiveTrackContext)

		_ = s.peerConnection.RemoveTrack(active.LoadSender())
		_ = active.trackContext.Close()

		s.MapDeleteTrack(id)
		return true
	})

	// for id, active := range s.tracks {
	// 	_ = s.peerConnection.RemoveTrack(active.LoadSender())
	// 	_ = active.trackContext.Close()
	// 	delete(s.tracks, id)
	// }

	return err
}

func (s *Subscriber) Done() <-chan struct{} {
	return s.ctx.Done()
}
