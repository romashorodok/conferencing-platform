package roomcontext

import (
	"context"
	"log"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pion/rtcp"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/peercontext"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/sfu"
	"github.com/romashorodok/conferencing-platform/pkg/controller/room"
)

type roomContext struct {
	sync.Mutex

	peerContextMap map[protocol.PeerID]protocol.PeerContext
	sfu            *sfu.SelectiveForwardingUnit
	roomID         protocol.RoomID

	logger    *slog.Logger
	webrtcAPI *webrtc.API
	ctx       context.Context
	cancel    context.CancelCauseFunc
}

// dispatchKeyFrame sends a keyframe to all PeerConnections, used everytime a new user joins the call
// func (r *roomContext) dispatchKeyFrame() {
// 	r.Lock()
// 	defer r.Unlock()
//
// 	for i := range r.peerContextMap {
// 		for _, receiver := range r.peerContextMap[i].GetPeerConnection().GetReceivers() {
// 			if receiver.Track() == nil {
// 				continue
// 			}
//
// 			_ = r.peerContextMap[i].GetPeerConnection().WriteRTCP([]rtcp.Packet{
// 				&rtcp.PictureLossIndication{
// 					MediaSSRC: uint32(receiver.Track().SSRC()),
// 				},
// 			})
// 		}
// 	}
// }

// func (r *roomContext) signalPeerContexts() {
// 	r.Lock()
// 	defer func() {
// 		r.Unlock()
// 		r.dispatchKeyFrame()
// 	}()
//
// 	attemptSync := func() (tryAgain bool) {
// 		for i := range r.peerContextMap {
// 			// log.Println("peer iteration", i)
// 			// log.Println(r.peerContextMap)
//
// 			if r.peerContextMap[i].GetPeerConnection().ConnectionState() == webrtc.PeerConnectionStateClosed {
// 				delete(r.peerContextMap, i)
// 				log.Println("peer context closed")
// 				return true
// 			}
//
// 			// map of sender we already are seanding, so we don't double send
// 			existingSenders := map[string]bool{}
//
// 			log.Println("peer senders", r.peerContextMap[i].GetPeerConnection().GetSenders())
//
// 			for _, sender := range r.peerContextMap[i].GetPeerConnection().GetSenders() {
// 				log.Println("sender", sender)
//
// 				if sender.Track() == nil {
// 					log.Println("peer dont have tracks")
// 					continue
// 				}
//
// 				existingSenders[sender.Track().ID()] = true
// 				log.Println(existingSenders)
//
// 				// If we have a RTPSender that doesn't map to a existing track remove and signal
//
// 				if _, ok := r.sfu.TrackLocalList[sender.Track().ID()]; !ok {
// 					if err := r.peerContextMap[i].GetPeerConnection().RemoveTrack(sender); err != nil {
// 						return true
// 					}
// 				}
//
// 				// Don't receive videos we are sending, make sure we don't have loopback
// 				for _, receiver := range r.peerContextMap[i].GetPeerConnection().GetReceivers() {
// 					log.Println("reciver", receiver)
//
// 					if receiver.Track() == nil {
// 						log.Println("Sefl signaling dont do it")
// 						continue
// 					}
//
// 					existingSenders[receiver.Track().ID()] = true
// 				}
//
// 				// Add all track we aren't sending yet to the PeerConnection
//
// 				for trackID := range r.sfu.TrackLocalList {
// 					if _, ok := existingSenders[trackID]; !ok {
// 						if _, err := r.peerContextMap[i].GetPeerConnection().AddTrack(r.sfu.TrackLocalList[trackID]); err != nil {
// 							return true
// 						}
// 					}
// 				}
//
// 				log.Println("existing senders", existingSenders)
//
// 				offer, err := r.peerContextMap[i].GetPeerConnection().CreateOffer(nil)
// 				log.Println("Have offer created for", i)
// 				if err != nil {
// 					return true
// 				}
//
// 				if err = r.peerContextMap[i].GetPeerConnection().SetLocalDescription(offer); err != nil {
// 					log.Println("Set local desc for peer. err", err)
// 					return true
// 				}
// 				log.Println("send offer", offer)
//
// 				offerString, err := json.Marshal(offer)
// 				if err != nil {
// 					return true
// 				}
//
// 				signalingDataChannel, err := r.peerContextMap[i].GetSignalingChannel()
// 				log.Println("signaling data channel", signalingDataChannel)
// 				if err != nil {
// 					return true
// 				}
//
// 				if err = signalingDataChannel.Send(offerString); err != nil {
// 					return true
// 				}
//
// 				// https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API/Perfect_negotiation
// 				// So my media server is always polite peer
//
// 				// offer, err := peerConnections[i].peerConnection.CreateOffer(nil)
// 				// if err != nil {
// 				// 	return true
// 				// }
// 				//
// 				// if err = peerConnections[i].peerConnection.SetLocalDescription(offer); err != nil {
// 				// 	return true
// 				// }
// 				//
// 				// offerString, err := json.Marshal(offer)
// 				// if err != nil {
// 				// 	return true
// 				// }
// 				//
// 				// if err = peerConnections[i].websocket.WriteJSON(&websocketMessage{
// 				// 	Event: "offer",
// 				// 	Data:  string(offerString),
// 				// }); err != nil {
// 				// 	return true
// 				// }
//
// 			}
// 		}
//
// 		return
// 	}
//
// 	for syncAttempt := 0; ; syncAttempt++ {
// 		if syncAttempt == 25 {
// 			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
// 			go func() {
// 				log.Println("Sync attempt", syncAttempt)
// 				time.Sleep(time.Second * 3)
// 				r.signalPeerContexts()
// 			}()
// 			return
// 		}
//
// 		if !attemptSync() {
// 			log.Println("Succes sync")
// 			break
// 		}
// 	}
// }

type peerConnectionState struct {
	peerConnection *webrtc.PeerConnection
	peerContext    protocol.PeerContext
}

var (
	listLock        sync.RWMutex
	peerConnections []peerConnectionState
)

func dispatchKeyFrame() {
	listLock.Lock()
	defer listLock.Unlock()

	for i := range peerConnections {
		for _, receiver := range peerConnections[i].peerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = peerConnections[i].peerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}

func (r *roomContext) signalPeerConnections() {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		dispatchKeyFrame()
	}()

	trackLocals := r.sfu.TrackLocalList

	log.Println("track locals", trackLocals)
	attemptSync := func() (tryAgain bool) {
		for i := range peerConnections {
			if peerConnections[i].peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				peerConnections = append(peerConnections[:i], peerConnections[i+1:]...)
				return true // We modified the slice, start from the beginning
			}

			// map of sender we already are seanding, so we don't double send
			existingSenders := map[string]bool{}

			for _, sender := range peerConnections[i].peerConnection.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				// If we have a RTPSender that doesn't map to a existing track remove and signal
				if _, ok := trackLocals[sender.Track().ID()]; !ok {
					if err := peerConnections[i].peerConnection.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}
			// log.Println(peerConnections[i].peerConnection.GetReceivers())

			// Don't receive videos we are sending, make sure we don't have loopback
			for _, receiver := range peerConnections[i].peerConnection.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}

				existingSenders[receiver.Track().ID()] = true
			}

			// Add all track we aren't sending yet to the PeerConnection
			for trackID := range trackLocals {
				if _, ok := existingSenders[trackID]; !ok {
					if _, err := peerConnections[i].peerConnection.AddTrack(trackLocals[trackID]); err != nil {
						return true
					}
				}
			}
			log.Println(existingSenders)

            <-webrtc.GatheringCompletePromise( peerConnections[i].peerConnection)

			// offer, err := peerConnections[i].peerConnection.CreateAnswer(nil)
			// if err != nil {
			// 	log.Println(err)
			// 	return true
			// }

			// if err = peerConnections[i].peerConnection.SetLocalDescription(offer); err != nil {
			// 	log.Println(err)
			// 	return true
			// }

			// offerString, err := json.Marshal(offer)
			// if err != nil {
			// 	return true
			// }
			//
			// signaling, err := peerConnections[i].peerContext.GetSignalingChannel()
			// if err != nil || signaling == nil {
			// 	log.Println("Signaling data channel empty")
			// 	return true
			// }
			// if err = signaling.SendText(string(offerString)); err != nil {
			// 	return true
			// }

			// if err = peerConnections[i].signaling.SendText(string(offerString)); err != nil {
			// 	return true
			// }
		}

		return
	}

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
			go func() {
				time.Sleep(time.Second * 3)
				r.signalPeerConnections()
			}()
			return
		}

		if !attemptSync() {
			log.Println("success signaling")
			break
		}
	}
}

func (r *roomContext) AddParticipant(offer string) (protocol.PeerContext, error) {
	r.Lock()
	defer r.Unlock()
	// go func() {
	// 	for range time.NewTicker(time.Second * 3).C {
	// 		r.dispatchKeyFrame()
	// 	}
	// }()

	peerID := uuid.New().String()
	peerContext, err := peercontext.NewPeerContext(
		peercontext.NewPeerContext_Params{
			API:           r.webrtcAPI,
			Logger:        r.logger,
			SFU:           r.sfu,
			ParentContext: r.ctx,
			PeerID:        peerID,
		},
	)
	if err != nil {
		return nil, err
	}

	if err := peerContext.SetRemoteSessionDescriptor(offer); err != nil {
		peerContext.Cancel(err)
		return nil, err
	}
	r.peerContextMap[peerID] = peerContext

	peerContext.OnDataChannel()
	peerContext.OnCandidate()
	peerContext.OnTrack(r.signalPeerConnections)

	// signaling, _ := peerContext.GetSignalingChannel()
	peerConnections = append(peerConnections,
		peerConnectionState{
			peerConnection: peerContext.GetPeerConnection(),
			peerContext:    peerContext,

			// signaling:      signaling,
		})

    peerConnection := peerContext.GetPeerConnection()
    peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate){
        log.Println("For", peerID, "ICE candidate fire", candidate)
    })

	return peerContext, nil
}

func (r *roomContext) Info() *room.Room {
	var participants []room.Participant

	for _, peerContext := range r.peerContextMap {
		info := peerContext.Info()

		participants = append(participants, room.Participant{
			Id:   &info.ID,
			Name: &info.Name,
		})
	}

	return &room.Room{
		Host:         new(string),
		Participants: &participants,
		Port:         new(string),
		SessionID:    &r.roomID,
	}
}

func (r *roomContext) Cancel(reason error) {
	r.cancel(reason)
}

var _ protocol.RoomContext = (*roomContext)(nil)

type RoomContextOption struct {
	WebrtcAPI  *webrtc.API
	Logger     *slog.Logger
	RoomID     protocol.RoomID
	RoomOption *protocol.RoomCreateOption
}

func NewRoomContext(roomOption RoomContextOption) *roomContext {
	ctx, cancel := context.WithCancelCause(context.TODO())

	return &roomContext{
		roomID:         roomOption.RoomID,
		peerContextMap: make(map[string]protocol.PeerContext),
		sfu:            sfu.NewSelectiveForwardUnit(),
		logger:         roomOption.Logger,
		webrtcAPI:      roomOption.WebrtcAPI,
		ctx:            ctx,
		cancel:         cancel,
	}
}
