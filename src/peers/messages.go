package peers

import (
	"encoding/binary"
	"fmt"
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
	if len(resp) > 0 && resp[len(resp)-1] != 1 {
		fmt.Println(len(resp))
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
