package peers

import (
	"io"
	"net"
)

func SendInterested(conn net.Conn) ([]byte, error) {
	msg := []byte{0, 0, 0, 1, 2}
	_, err := conn.Write(msg)
	if err != nil {
		return nil, err
	}

	resp, err := AwaitResponse(conn, 5)
	return resp, err
}

func AwaitResponse(conn net.Conn, size uint32) ([]byte, error) {
	resp := make([]byte, size)
	_, err := io.ReadFull(conn, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
