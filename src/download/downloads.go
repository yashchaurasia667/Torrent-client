package download

import (
	"bytes"
	"fmt"
	"net"
	"torrent-client/parser"
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
		changeBit := byte(1 << ind)
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
	for i := range downloaded {
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

func DownloadPiece(conn net.Conn, bitfield []byte, downloaded []byte, pieceLength uint64, piecesHash [][]byte) ([]byte, error) {
	begin := uint32(0)
	// TODO: fix the piece size to be smaller for the last piece if needed
	piece := make([]byte, pieceLength)

	dIndex, bIndex, err := GetNextDownloadablePiece(bitfield, downloaded)
	if err != nil {
		return nil, err
	}
	pieceIndex := uint32(dIndex*8 + bIndex)

	for {
		block, err := peers.RequestPiece(conn, pieceIndex, begin, BLOCK_SIZE)
		if err != nil {
			return nil, err
		}

		copy(piece[begin:begin+BLOCK_SIZE], block[13:])
		begin += BLOCK_SIZE
		if begin == uint32(pieceLength) {
			break
		}
	}
	fmt.Println("Downloaded piece index:", pieceIndex)
	downloaded[dIndex] += byte(1 << bIndex)
	fmt.Println(downloaded[dIndex])

	// verify the downloaded piece
	expected := piecesHash[pieceIndex]
	computed := parser.GetSha1Hash(piece)

	if !bytes.Equal(computed, expected) {
		fmt.Println("expected ", expected, "got", computed)
		return nil, fmt.Errorf("piece hash mismatch")
	}

	return piece, nil
}
