package peers

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
	"torrent-client/src/parser"
	"torrent-client/src/utils"
)

/*
MESSAGE STRUCTURE
TOTAL LENGTH -> 68 bytes
{
	0: length of protocol string, 19 in this case
	1: protocol string, "BitTorrent Protocol" in this case
	20: 8 0 bytes, make([]byte, 8)
	28: info hash of the torrent
	48: your peer id
}
*/

const PROTOCOL_STRING = "BitTorrent protocol"
const HANDSHAKE_TIMEOUT = 10

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
	} else if !bytes.Equal(resp[28:48], infoHash) {
		return fmt.Errorf("info hash mismatch")
	} else {
		return nil
	}
}

func PerformHandshake(peer parser.Peer, infoHash []byte, peerId []byte, downloaded *utils.Downloaded) (*parser.Peer, error) {
	dest := net.JoinHostPort(peer.Ip.String(), strconv.FormatUint(uint64(peer.Port), 10))
	// fmt.Println("Connecting to", dest)

	conn, err := net.DialTimeout("tcp", dest, HANDSHAKE_TIMEOUT*time.Second)
	if err != nil {
		return nil, err
	}

	// Send handshake
	handshake := buildHandshake(infoHash, peerId)
	_, err = conn.Write(handshake)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Receive handshake
	resp := make([]byte, 68)
	_, err = io.ReadFull(conn, resp)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Validate response
	if err := validateResponse(resp, infoHash); err != nil {
		fmt.Println("Error: ", err)
		// fmt.Println("Given infoHash: ", infoHash)
		// fmt.Println("Received infohash", resp[20:48])
		conn.Close()
		return nil, err
	}

	// Send my own bitfield
	if downloaded.GetPieceCount() > 0 {
		fmt.Println("Sending own bitfield", downloaded.GetContent())
		err := SendBitfield(downloaded.GetContent(), conn)
		if err != nil {
			return nil, err
		}
	}

	// Get bitfield message
	msg, err := AwaitResponse(conn, 5)
	if err != nil || msg[len(msg)-1] != 5 {
		return nil, err
	}

	bitf, err := AwaitResponse(conn, ReadLength(msg))
	if err != nil {
		return nil, err
	}

	// fmt.Println("Connected to peer", peer.Ip)
	return &parser.Peer{Ip: peer.Ip, Port: peer.Port, Conn: conn, PeerId: [20]byte(resp[48:]), Bitfield: bitf}, nil
}

// func StartPeerConnections(peers []parser.Peer, infoHash []byte, peerId []byte) ([]ConnectedPeer, error) {
// 	var available_peers []ConnectedPeer
// 	for _, peer := range peers {
// 		resp, ip, err := PerformHandshake(peer, infoHash, peerId)
// 		if err != nil {
// 			continue
// 		}

// 		available_peers = append(available_peers, ConnectedPeer{Ip: ip, PeerId: [20]byte(resp[48:])})
// 	}
// 	return available_peers, nil
// }
