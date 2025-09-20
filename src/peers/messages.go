package peers

import (
	"encoding/binary"
	"io"
	"net"
)

/*
MESSAGE IDS
0 -> choke
1 -> unchoke
2 -> interested
3 -> not interested
4 -> have
5 -> bitfield
6 -> request
7 -> piece
8 -> cancel
*/

func ReadLength(length []byte) uint32 {
	return binary.BigEndian.Uint32(length) - 1
}

func CheckInterested(conn net.Conn) bool {
	msg := []byte{0, 0, 0, 1, 2}
	_, err := conn.Write(msg)
	if err != nil {
		return false
	}

	resp, err := AwaitResponse(conn, 5)
	if (len(resp) > 0 && resp[len(resp)-1] != 1) || err != nil {
		return false
	}
	return true
}

func AwaitResponse(conn net.Conn, size uint32) ([]byte, error) {
	resp := make([]byte, size)
	_, err := io.ReadFull(conn, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func RequestPiecec(conn net.Conn, pieceIndex uint32, begin uint32, blockLength uint32) ([]byte, error) {
	msg := make([]byte, 17)
	binary.BigEndian.PutUint32(msg[0:4], 13)
	msg[4] = 6
	binary.BigEndian.PutUint32(msg[5:9], pieceIndex)
	binary.BigEndian.PutUint32(msg[9:13], begin)
	binary.BigEndian.PutUint32(msg[13:17], blockLength)

	_, err := conn.Write(msg)
	if err != nil {
		return nil, err
	}

	resp, err := AwaitResponse(conn, 13+blockLength)
	if err != nil {
		return nil, err
	}

	if err = SendHavePiece(conn, pieceIndex); err != nil {
		return nil, err
	}

	return resp, nil
}

func SendHavePiece(conn net.Conn, pieceIndex uint32) error {
	msg := make([]byte, 9)
	binary.BigEndian.PutUint32(msg[0:4], 5)
	msg[4] = 4
	binary.BigEndian.PutUint32(msg[5:9], pieceIndex)

	_, err := conn.Write(msg)
	if err != nil {
		return err
	}

	return nil
}
