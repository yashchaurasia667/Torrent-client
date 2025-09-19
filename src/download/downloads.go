package download

import (
	"fmt"
	"math"
)

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
			return -1
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

func GetNextDownloadablePiece(bitfield []byte, downloaded []byte) (uint32, error) {
	if len(bitfield) != len(downloaded) {
		return 0, fmt.Errorf("piece count received from peer is not same as parsed count. expected %d got %d", len(downloaded), len(bitfield))
	}

	var index uint32 = 0
	for i := range len(downloaded) {
		if downloaded[i] == 255 {
			continue
		} else if downloaded[i] != bitfield[i] {
			ind := getIndex(bitfield[i], downloaded[i])
			if ind != -1 {
				index = uint32(8*i + ind)
				break
			}
		}
	}

	return index, nil
}
