package download

import "fmt"

func getFirstEnabledBit(a byte) byte {
	index := byte(7)
	for {
		diff := (a >> index) & 1
		if diff == 1 {
			break
		}
		index--
	}
	return index
}

func GetFirstDisabledBit(a byte) int {
	index := 7
	for {
		diff := (a >> 7) & 1
		if diff == 0 {
			break
		} else if diff != 0 && index == 0 {
			return -1
		}
		index--
	}
	return index
}

func getIndex(bitfield byte, downloaded byte) {
	// GET THE INDEX OF THE FIRST ONE IN BITFIELD
	// ind := getFirstEnabledBit(bitfield)
	// CHECK IF THAT BIT IS SET IN DOWNLAODED
	// diff :=
	// IF SO GET THE NEXT BIT THAT'S SET IN BITFIELD ELSE RETURN INDEX
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
