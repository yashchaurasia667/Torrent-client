package download

import (
	"fmt"
	"math"
	"net"
	"torrent-client/peers"
)

const BLOCK_SIZE uint32 = 16384 // 16 kib

func GetFirstEnabledBit(a byte) int {
	index := 7
	for {
		diff := (a >> index) & 1
		if diff == 1 {
			break
		} else if diff != 1 && index == 0 {
			return -1
		}
		index--
	}
	return index
}

func GetFirstDisabledBit(a byte) int {
	index := 7
	for {
		diff := (a >> index) & 1
		if diff == 0 {
			break
		} else if diff != 0 && index == 0 {
			return -1
		}
		index--
	}
	return index
}

func getIndex(bitfield byte, downloaded byte) int {
	// GET THE INDEX OF THE FIRST ZERO IN DOWNLOADED
	for {
		ind := GetFirstDisabledBit(downloaded)
		if ind == -1 {
			return ind
		}
		// CHECK IF THAT BIT IS SET IN BITFIELD IF SO GET THE NEXT BIT THAT'S SET IN BITFIELD
		changeBit := byte(math.Pow(2.0, float64(ind)))
		diff := (bitfield >> ind) & changeBit
		if diff == 1 {
			return ind
		}
		// ELSE SET THE ZERO BIT TO ONE AND REPEAT
		downloaded += changeBit
	}
}

func GetNextDownloadablePiece(bitfield []byte, downloaded []byte) (int, int, error) {
	if len(bitfield) != len(downloaded) {
		return 0, 0, fmt.Errorf("piece count received from peer is not same as parsed count. expected %d got %d", len(downloaded), len(bitfield))
	}

	var downloadIndex int = -1
	var bitIndex int = 0
	for i := range len(downloaded) {
		if downloaded[i] == 255 {
			continue
		} else if downloaded[i] != bitfield[i] {
			bitIndex = getIndex(bitfield[i], downloaded[i])
			if bitIndex != -1 {
				downloadIndex = i
				break
			}
		}
	}

	if downloadIndex == -1 && bitIndex == -1 {
		return 0, 0, fmt.Errorf("no downloadable piece found")
	}

	return downloadIndex, bitIndex, nil
}

func DownloadPiece(conn net.Conn, bitfield []byte, downloaded []byte, pieceLength uint64) ([]byte, error) {
	begin := uint32(0)
	var piece []byte
	dIndex, bIndex, err := GetNextDownloadablePiece(bitfield, downloaded)
	if err != nil {
		return nil, err
	}
	pieceIndex := uint32(dIndex*8 + bIndex)

	for {
		block, err := peers.RequestPiecec(conn, pieceIndex, begin, BLOCK_SIZE)
		if err != nil {
			return nil, err
		}
		// fmt.Println("Got a block! begin: ", begin)

		piece = append(piece, block...)
		begin += BLOCK_SIZE
		if begin >= uint32(pieceLength) {
			break
		}
	}

	return piece, nil
}
