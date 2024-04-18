package sfu

import "errors"

var (
	ErrPeerConnectionClosed         = errors.New("peerConnection is closed")
	ErrSignalRetry                  = errors.New("unable dispatch signal offer")
	ErrRoomAlreadyExists            = errors.New("room already exists")
	ErrRoomNotExist                 = errors.New("room not exist")
	ErrRoomIDIsEmpty                = errors.New("room id is empty")
	ErrRoomCancelByUser             = errors.New("room canceled by user")
	ErrTrackCancelByUser            = errors.New("track canceled by user")
	ErrTrackAlreadyExists           = errors.New("track already exists")
	ErrTrackNotFound                = errors.New("track not found")
	ErrUnsupportedTrack             = errors.New("track codec is not supported")
	ErrDescNotFound                 = errors.New("session desc not found")
	ErrEmptyPipelinesArg            = errors.New("require at least one pipeline")
	ErrInvalidPipelineAllocatorName = errors.New("Invalid pipeline allocator name")

	// ** Subscriber
	ErrWatchTrackDetachNotFound       = errors.New("subscriber track not found. ")
	ErrWatchTrackDetachSenderNotFound = errors.New("subscriber sender not found. ")

	// ** ActiveTrackContext
	ErrSwitchActiveTrackNotFoundSender      = errors.New("active track context not found sender")
	ErrSwitchActiveTrackNotFoundTransiv     = errors.New("active track context not found transiver")
	ErrSwitchActiveTrackUnableCreateTransiv = errors.New("unable re-create new transiver")

	// ** SessionDesc
	ErrSubmitEmptyPendingSessionDesc = errors.New("don't have pending session desc to submit. ")
	ErrSubmitOfferStateEmpty         = errors.New("passed empty offer hash")
	ErrSubmitOfferRaceCondition      = errors.New("submit offer race condition found")

	// ** TransceiverPool
	ErrNotFoundTransceiver = errors.New("not found transceiver")
)
