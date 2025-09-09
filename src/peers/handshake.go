package peers

import (
	"bytes"
	"fmt"
	"io"
	"net"

	// "os"
	"strconv"
	"time"
	"torrent-client/parser"
)

const PROTOCOL_STRING = "BitTorrent protocol"

type connectedPeer struct {
	Ip     net.IP
	PeerId [20]byte
}

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

func validateResponse(resp []byte, infoHash []byte) error {
	if int(resp[0]) != len(PROTOCOL_STRING) {
		return fmt.Errorf("bitfield mismatch")
	} else if string(resp)[1:20] != PROTOCOL_STRING {
		return fmt.Errorf("protocol mismatched expected %s got %s", PROTOCOL_STRING, string(resp[1:20]))
	} else if !bytes.Equal(resp[20:48], infoHash) {
		return fmt.Errorf("info hash mismatch")
	} else {
		return nil
	}
}

func PerformHandshake(peer parser.Peer, infoHash []byte, peerId []byte) ([]byte, net.IP, error) {
	dest := net.JoinHostPort(peer.Ip.String(), strconv.FormatUint(uint64(peer.Port), 10))

	conn, err := net.DialTimeout("tcp", dest, 5*time.Second)
	if err != nil {
		return nil, nil, err
	}
	// TODO: DON'T CLOSE THE CONNECTION AFTER RECEIVING THE HANDSHAKE BUT INSTANTLY MOVE TO DOWNLOADING PEICES
	defer conn.Close()

	// Send handshake
	handshake := buildHandshake(infoHash, peerId)
	_, err = conn.Write(handshake)
	if err != nil {
		return nil, nil, err
	}

	// Receive handshake
	resp := make([]byte, 68)
	_, err = io.ReadFull(conn, resp)
	if err != nil {
		return nil, nil, err
	}

	// Validate response
	if err := validateResponse(resp, infoHash); err != nil {
		return nil, nil, err
	}

	fmt.Println("Connected to peer", peer.Ip)
	return resp[:], peer.Ip, nil
}

func StartPeerConnections(peers []parser.Peer, infoHash []byte, peerId []byte) error {
	var available_peers []connectedPeer
	for _, peer := range peers {
		resp, ip, err := PerformHandshake(peer, infoHash, peerId)
		if err != nil {
			// fmt.Fprintln(os.Stderr, "Handshake failed", err)
			continue
		}
		if err = validateResponse(resp, infoHash); err != nil {
			return err
		}
		available_peers = append(available_peers, connectedPeer{Ip: ip, PeerId: [20]byte(resp[48:])})
	}
	return nil
}
