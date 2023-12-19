package protocol

type PeerID = string

type PeerInfo struct {
	ID   string
	Name string
}

type PeerContext interface {
	SetRemoteSessionDescriptor(offer string) error
	GenerateSDPAnswer() (string, error)
	OnDataChannel()
	Info() *PeerInfo
}
