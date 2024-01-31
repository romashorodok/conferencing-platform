package room

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/sdp/v3"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/protocol"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/rtpstats"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/twcc"
	"github.com/romashorodok/conferencing-platform/pkg/controller/room"
	"go.uber.org/fx"
)

var (
	ErrPeerConnectionClosed = errors.New("peerConnection is closed")
	ErrRoomAlreadyExists    = errors.New("room already exists")
	ErrRoomNotExist         = errors.New("room not exist")
	ErrRoomIDIsEmpty        = errors.New("room id is empty")
	ErrRoomCancelByUser     = errors.New("room canceled by user")
)

type TrackCongestionControlWritable interface {
	WriteRTP(p *rtp.Packet) error
	OnFeedback(func(pkts []rtcp.Packet))
}

type TrackContext struct {
	track   *webrtc.TrackLocalStaticRTP
	sender  *webrtc.RTPSender
	twcc    *twcc.Responder
	twccExt uint8
}

func (t *TrackContext) WriteRTP(p *rtp.Packet) error {
	go func() {
		if t.twccExt <= 0 {
			return
		}

		ext := p.GetExtension(t.twccExt)
		if ext == nil {
			return
		}

		t.twcc.Push(binary.BigEndian.Uint16(ext[0:2]), time.Now().UnixNano(), p.Marker)
	}()
	return t.track.WriteRTP(p)
}

func (t *TrackContext) Close() (err error) {
	err = t.sender.ReplaceTrack(nil)
	err = t.sender.Stop()
	return
}

func (t *TrackContext) OnFeedback(f func(pkts []rtcp.Packet)) {
	t.twcc.OnFeedback(f)
}

type NewTrackContextParams struct {
	Track    *webrtc.TrackLocalStaticRTP
	Sender   *webrtc.RTPSender
	TWCC_EXT uint8
	SSRC     uint32
}

func NewTrackContext(params NewTrackContextParams) *TrackContext {
	return &TrackContext{
		track:   params.Track,
		sender:  params.Sender,
		twcc:    twcc.NewTransportWideCCResponder(params.SSRC),
		twccExt: params.TWCC_EXT,
	}
}

type LoopbackTrackContext struct {
	*TrackContext
}

type Subscriber struct {
	peerConnection *webrtc.PeerConnection
	peerId         string

	sid string

	loopback map[string]*LoopbackTrackContext
	tracks   map[string]*TrackContext
	tracksMu sync.Mutex
}

func (s *Subscriber) Close() (err error) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	for id, t := range s.tracks {
		err = t.Close()
		err = s.peerConnection.RemoveTrack(t.sender)
		delete(s.tracks, id)
	}

	for id, t := range s.loopback {
		err = t.Close()
		err = s.peerConnection.RemoveTrack(t.sender)
		delete(s.loopback, id)
	}

	return err
}

func (s *Subscriber) HasTrack(trackID string) (*TrackContext, bool) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	track, exist := s.tracks[trackID]
	return track, exist
}

func (s *Subscriber) CreateTrack(t *webrtc.TrackRemote, recv *webrtc.RTPReceiver) (*TrackContext, error) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()

	if track, exist := s.tracks[t.ID()]; exist {
		if track != nil {
			return track, nil
		}
	}

	// NOTE: Track may have same id, but it may have different layerID(RID)
	track, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		log.Println("unable create track for subscriber. track", err)
		return nil, err
	}

	sender, err := s.peerConnection.AddTrack(track)
	if err != nil {
		log.Println("uanble add track to the subscriber. sender", err)
		return nil, err
	}

	var twccExt uint8
	for _, fb := range t.Codec().RTCPFeedback {
		switch fb.Type {
		case webrtc.TypeRTCPFBGoogREMB:
		case webrtc.TypeRTCPFBNACK:
			log.Printf("Unsupported rtcp feedbacak %s type", fb.Type)
			continue

		case webrtc.TypeRTCPFBTransportCC:
			if strings.HasPrefix(t.Codec().MimeType, "video") {
				for _, ext := range recv.GetParameters().HeaderExtensions {
					if ext.URI == sdp.TransportCCURI {
						twccExt = uint8(ext.ID)
						break
					}
				}
			}
		}
	}

	trackContext := NewTrackContext(NewTrackContextParams{
		Track:    track,
		Sender:   sender,
		TWCC_EXT: twccExt,
		SSRC:     uint32(t.SSRC()),
	})
	s.tracks[t.ID()] = trackContext
	return trackContext, nil
}

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

func (s *Subscriber) LoopbackTrack(t *webrtc.TrackRemote, recv *webrtc.RTPReceiver) (*LoopbackTrackContext, error) {
	trackCtx, err := s.CreateTrack(t, recv)
	if err != nil {
		return nil, err
	}

	s.loopback[trackCtx.track.ID()] = &LoopbackTrackContext{
		TrackContext: trackCtx,
	}

	return s.loopback[trackCtx.track.ID()], nil
}

func (s *Subscriber) GetLoopbackTrack(trackID string) (track *LoopbackTrackContext, exist bool) {
	s.tracksMu.Lock()
	defer s.tracksMu.Unlock()
	track, exist = s.loopback[trackID]
	return
}

type PeerContext struct {
	Ctx    context.Context
	Cancel context.CancelCauseFunc

	peerID         protocol.PeerID
	webrtc         *webrtc.API
	peerConnection *webrtc.PeerConnection
	stats          *rtpstats.RtpStats
	ws             *threadSafeWriter
	signalMu       sync.Mutex
	Subscriber     *Subscriber
}

func (p *PeerContext) signalPeerConnection() (bool, error) {
	if p.peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
		return false, ErrPeerConnectionClosed
	}

	offer, err := p.peerConnection.CreateOffer(nil)
	if err != nil {
		return false, err
	}

	if err = p.peerConnection.SetLocalDescription(offer); err != nil {
		return false, err
	}

	offerString, err := json.Marshal(offer)
	if err != nil {
		return false, err
	}

	if err = p.ws.WriteJSON(&websocketMessage{
		Event: "offer",
		Data:  string(offerString),
	}); err != nil {
		return false, err
	}
	return true, nil
}

func (p *PeerContext) SignalPeerConnection() {
	p.signalMu.Lock()
	defer p.signalMu.Unlock()

	signal := debounce(p.signalPeerConnection, time.Millisecond*10)

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt >= 25 {
			go func() {
				time.Sleep(time.Millisecond * 60)
				signal()
			}()
			break
		}
		success, err := signal()
		if !errors.Is(err, ErrPeerConnectionClosed) || success {
			break
		}
		log.Printf("Signaling for %s. Err %s", p.peerID, err)
	}
}

func (p *PeerContext) NewSubscriber() {
	subscriber := &Subscriber{
		peerId:         p.peerID,
		peerConnection: p.peerConnection,
		loopback:       make(map[string]*LoopbackTrackContext),
		tracks:         make(map[string]*TrackContext),
	}
	p.Subscriber = subscriber
}

func (p *PeerContext) NewPeerConnection() error {
	peerConnection, err := p.webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return err
	}
	p.peerConnection = peerConnection
	return nil
}

func (p *PeerContext) Close() error {
	return p.peerConnection.Close()
}

type NewPeerContextParams struct {
	Parent context.Context
	WS     *threadSafeWriter
	API    *webrtc.API
}

func NewPeerContext(params NewPeerContextParams) *PeerContext {
	ctx, cancel := context.WithCancelCause(params.Parent)
	return &PeerContext{
		peerID: uuid.NewString(),
		Ctx:    ctx,
		Cancel: cancel,
		ws:     params.WS,
		webrtc: params.API,
	}
}

func debounce(fn func() (bool, error), delay time.Duration) func() (bool, error) {
	var (
		mu    sync.Mutex
		last  time.Time
		timer *time.Timer
	)
	return func() (result bool, err error) {
		mu.Lock()
		defer mu.Unlock()

		if timer != nil {
			timer.Stop()
		}

		elapsed := time.Since(last)
		if elapsed > delay {
			result, err = fn()
			last = time.Now()
			return
		}

		timer = time.AfterFunc(delay-elapsed, func() {
			mu.Lock()
			defer mu.Unlock()
			result, err = fn()
			last = time.Now()
		})

		return result, err
	}
}

type PeerContextPool struct {
	subscriberMu sync.Mutex
	pool         map[string]*PeerContext
}

func (s *PeerContextPool) SignalPeerContexts() {
	for _, peerContext := range s.pool {
		peerContext.SignalPeerConnection()
	}
}

func (s *PeerContextPool) Get() []*PeerContext {
	s.subscriberMu.Lock()
	defer s.subscriberMu.Unlock()

	var result []*PeerContext
	for _, sub := range s.pool {
		result = append(result, sub)
	}

	return result
}

func (s *PeerContextPool) Add(sub *PeerContext) error {
	s.subscriberMu.Lock()
	defer s.subscriberMu.Unlock()

	if _, exist := s.pool[sub.peerID]; exist {
		return errors.New("Subscriber exist. Remove it first")
	}

	s.pool[sub.peerID] = sub
	return nil
}

func (s *PeerContextPool) Remove(sub *PeerContext) (err error) {
	s.subscriberMu.Lock()
	defer s.subscriberMu.Unlock()

	if sub == nil {
		return
	}

	if s, exist := s.pool[sub.peerID]; exist {
		err = s.Close()
	}

	delete(s.pool, sub.peerID)
	return err
}

func NewSubscriberPool() *PeerContextPool {
	return &PeerContextPool{
		pool: make(map[string]*PeerContext),
	}
}

type RoomNotifier struct {
	listeners     map[string]*threadSafeWriter
	updateRoomCh  chan struct{}
	updateRoomsMu sync.Mutex
}

func (n *RoomNotifier) Listen(id string, w *threadSafeWriter) {
	n.updateRoomsMu.Lock()
	defer n.updateRoomsMu.Unlock()
	n.listeners[id] = w
}

func (n *RoomNotifier) Stop(id string) {
	delete(n.listeners, id)
}

func (n *RoomNotifier) DispatchUpdateRooms() {
	n.updateRoomsMu.Lock()
	defer n.updateRoomsMu.Unlock()

	if len(n.listeners) == 0 {
		return
	}

	n.updateRoomCh <- struct{}{}
}

func (n *RoomNotifier) getListeners() (result []*threadSafeWriter) {
	for _, listener := range n.listeners {
		result = append(result, listener)
	}
	return
}

func (n *RoomNotifier) OnUpdateRooms(ctx context.Context, fn func(*threadSafeWriter)) {
	var threshold uint64 = 1000000
	var step uint64 = 2
	for {
		select {
		case <-ctx.Done():
			return
		case <-n.updateRoomCh:
			ParallelExec(n.getListeners(), threshold, step, fn)
		}
	}
}

func NewRoomNotifier() *RoomNotifier {
	return &RoomNotifier{
		listeners:    make(map[string]*threadSafeWriter),
		updateRoomCh: make(chan struct{}),
	}
}

type roomContext struct {
	roomID          protocol.RoomID
	peerContextPool *PeerContextPool
}

func (r *roomContext) Info() room.Room {
	participants := make([]room.Participant, 0)

	for _, p := range r.peerContextPool.Get() {
		participants = append(participants, room.Participant{
			Id: p.peerID,
		})
	}

	return room.Room{
		RoomId:       r.roomID,
		Participants: participants,
	}
}

type NewRoomContextParams struct {
	RoomID protocol.RoomID
}

func NewRoomContext(params NewRoomContextParams) *roomContext {
	return &roomContext{
		roomID:          params.RoomID,
		peerContextPool: NewSubscriberPool(),
	}
}

type RoomService struct {
	sync.Mutex

	webrtcAPI      *webrtc.API
	logger         *slog.Logger
	roomContextMap map[protocol.RoomID]*roomContext
	roomNotifier   *RoomNotifier
}

func (s *RoomService) GetRoom(roomID string) *roomContext {
	room, exist := s.roomContextMap[roomID]
	if !exist {
		return nil
	}
	return room
}

func (s *RoomService) ListRoom() []room.Room {
	result := make([]room.Room, 0)
	for _, room := range s.roomContextMap {
		result = append(result, room.Info())
	}
	return result
}

//
// func (s *roomService) DeleteRoom(roomID string) error {
// 	room, exist := s.roomContextMap[roomID]
// 	if !exist {
// 		return ErrRoomNotExist
// 	}
// 	room.Cancel(ErrRoomCancelByUser)
// 	delete(s.roomContextMap, roomID)
// 	return nil
// }

func NullableRoomID(roomID *string) string {
	if roomID != nil && *roomID != "" {
		return *roomID
	}
	return uuid.NewString()
}

func (s *RoomService) CreateRoom(option *protocol.RoomCreateOption) (*roomContext, error) {
	s.Lock()
	defer s.Unlock()

	roomID := NullableRoomID(option.RoomID)
	if _, exist := s.roomContextMap[roomID]; exist {
		return nil, ErrRoomAlreadyExists
	}

	s.roomContextMap[roomID] = NewRoomContext(NewRoomContextParams{
		RoomID: roomID,
	})

	room, exist := s.roomContextMap[roomID]
	if !exist && room == nil {
		return nil, errors.New("not found room or it's nil")
	}

	s.roomNotifier.DispatchUpdateRooms()

	return room, nil
}

type NewRoomServiceParams struct {
	fx.In

	WebrtcAPI    *webrtc.API
	Logger       *slog.Logger
	RoomNotifier *RoomNotifier
}

func NewRoomService(params NewRoomServiceParams) *RoomService {
	return &RoomService{
		webrtcAPI:      params.WebrtcAPI,
		logger:         params.Logger,
		roomContextMap: make(map[string]*roomContext),
		roomNotifier:   params.RoomNotifier,
	}
}
