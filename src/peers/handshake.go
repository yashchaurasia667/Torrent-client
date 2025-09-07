package peers

import (
	"fmt"
	"io"
	"net"
	// "os"
	"strconv"
	"time"
	"torrent-client/parser"
)

const PROTOCOL_STRING = "BitTorrent protocol"

func buildHandshake(infoHash []byte, peerId []byte) []byte {
	handshake := make([]byte, 68)

	// Length of the protocol string "BitTorrent protocol"
	handshake[0] = 19
	copy(handshake[1:], []byte(PROTOCOL_STRING))
	copy(handshake[20:], make([]byte, 8))
	copy(handshake[28:], infoHash[:])
	copy(handshake[48:], peerId[:])
	return handshake
}

func PerformHandshake(peer parser.Peer, infoHash []byte, peerId []byte) ([]byte, error) {
	dest := net.JoinHostPort(peer.Ip.String(), strconv.FormatUint(uint64(peer.Port), 10))
	// fmt.Println("Connecting to", dest)

	conn, err := net.DialTimeout("tcp", dest, 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	handshake := buildHandshake(infoHash, peerId)
	_, err = conn.Write(handshake)
	if err != nil {
		return nil, err
	}
	// fmt.Println("Handshake sent")

	resp := make([]byte, 68)
	_, err = io.ReadFull(conn, resp)
	if err != nil {
		return nil, err
	}
	// fmt.Println("Handshake received")

	if int(resp[0]) != len(PROTOCOL_STRING) || string(resp[1:20]) != PROTOCOL_STRING {
		return nil, fmt.Errorf("invalid protocol string")
	}

	var recvHash [20]byte
	copy(recvHash[:], resp[28:48])
	if recvHash != [20]byte(infoHash) {
		return nil, fmt.Errorf("infohash mismatch")
	}

	fmt.Println("Connected to peer", peer.Ip)
	// recvPeerId := resp[48:68]
	// fmt.Println("Connected to peer", recvPeerId)
	// fmt.Println("Handshake successful")
	return recvHash[:], nil
}

func StartPeerConnections(peers []parser.Peer, infoHash []byte, peerId []byte) error {
	for _, peer := range peers {
		recvHash, err := PerformHandshake(peer, infoHash, peerId)
		if err != nil {
			// fmt.Fprintln(os.Stderr, "Handshake failed", err)
			continue
		}

		fmt.Println(recvHash)
	}
	return nil
}
