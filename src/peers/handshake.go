package peers

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"
	"torrent-client/parser"
)

type initTCPconn struct {
	host net.IP
	port uint32
}

type HandShakeConn struct {
	Length     uint32
	Identifier []byte
	Reserved   []byte
	InfoHash   []byte
	PeerId     []byte
}

func PerformHandshake(peer parser.Peer, infoHash []byte, peerId []byte) error {
	dest := net.JoinHostPort(peer.Ip.String(), strconv.FormatUint(uint64(peer.Port), 10))
	fmt.Println("Connecting to ", dest)

	conn, err := net.DialTimeout("tcp", dest, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	handshake := HandShakeConn{
		Length:   19,
		Reserved: make([]byte, 8),
	}
	copy(handshake.Identifier, "Bittorrent Protocol")
	copy(handshake.InfoHash, infoHash)
	copy(handshake.PeerId, peerId)

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, handshake); err != nil {
		return err
	}
	if _, err := conn.Write(buf.Bytes()); err != nil {
		fmt.Println("Error writing connection for peer ", dest)
		return err
	}

	return nil
}

func StartPeerConnections(peers []parser.Peer, infoHash []byte, peerId []byte) error {
	for _, peer := range peers {
		// lis, err := net.Listen("tcp", peer.Ip.String()+":"+strconv.FormatUint(uint64(peer.Port), 10))
		// if err != nil {
		// 	return err
		// }
		// fmt.Println(lis)

		err := PerformHandshake(peer, infoHash, peerId)
		if err != nil {
			return err
		}
	}
	return nil
}
