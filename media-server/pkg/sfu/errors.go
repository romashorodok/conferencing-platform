package sfu

import "errors"

var (
	ErrPeerConnectionClosed  = errors.New("peerConnection is closed")
	ErrRoomAlreadyExists     = errors.New("room already exists")
	ErrRoomNotExist          = errors.New("room not exist")
	ErrRoomIDIsEmpty         = errors.New("room id is empty")
	ErrRoomCancelByUser      = errors.New("room canceled by user")
	ErrTrackCancelByUser     = errors.New("track canceled by user")
	ErrTrackAlreadyExists    = errors.New("track already exists")
	ErrUnsupportedTrackCodec = errors.New("track codec is not supported")

	ErrEmptyPipelinesArg            = errors.New("require at least one pipeline")
	ErrInvalidPipelineAllocatorName = errors.New("Invalid pipeline allocator name")
)
