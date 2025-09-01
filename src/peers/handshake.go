package peers

type Handshake struct {
	Length    uint64
	Protocol  []byte
	Reserverd []byte
	InfoHash  [20]byte
	PeerId    []byte
}

func hShake(infoHash [20]byte, peerId string) {
	hs := Handshake{
		Length:    19,
		Protocol:  []byte("BitTorrent protocol"),
		Reserverd: make([]byte, 8),
		InfoHash:  infoHash,
		PeerId:    []byte(peerId),
	}
}
