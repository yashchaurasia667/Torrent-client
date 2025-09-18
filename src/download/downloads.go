package download

import "fmt"

func getDifferentBit(a byte, b byte) uint32 {
	
}

func GetNextDownloadablePiece(bitfield []byte, downloaded []byte) (uint32, error) {
	if len(bitfield) != len(downloaded) {
		return 0, fmt.Errorf("piece count received from peer is not same as parsed count. expected %d got %d", len(downloaded), len(bitfield))
	}

	var index uint32 = 0
	for i := range len(downloaded) {
		if downloaded[i] != bitfield[i] {
		}
	}

	return index, nil
}
