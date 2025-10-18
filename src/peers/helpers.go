package peers

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"torrent-client/src/parser"
)

const PORT = 6881
const INIT = "BT"
const VERSION = "0003"
const NUM_PEERS = 50
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type ConnectionRequest struct {
	ProtocolId    uint64
	Action        uint32
	TransactionId uint32
}

type ConnectionResponse struct {
	Action        uint32
	TransactionId uint32
	ConnectionId  uint64
}

type HttpConnection struct {
	client http.Client
	peerId string
}

type AnnounceRequest struct {
	ConnectionId  uint64
	Action        uint32
	TransactionId uint32
	InfoHash      [20]byte
	PeerId        [20]byte
	Downloaded    uint64
	Left          uint64
	Uploaded      uint64
	Event         uint32
	IpAddress     uint32
	Key           uint32
	NumWant       int32
	Port          uint16
}

type AnnounceResponse struct {
	Action        uint32
	TransactionId uint32
	Interval      uint32
	Leechers      uint32
	Seeders       uint32
	Peers         []parser.Peer
}

func GetPeerId() string {
	return connection.peerId
}

func generatePeerId() string {
	prefix := "-" + INIT + VERSION
	b := make([]byte, 13)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return prefix + string(b)
}

func generateTransactionId() uint32 {
	var b [4]byte
	crand.Read(b[:])
	return binary.BigEndian.Uint32(b[:])
}

func buildTrackerUrl(trakerAddr string, infoHash []byte, totalLength uint64, connection *HttpConnection) (string, error) {
	u, err := url.Parse(trakerAddr)
	if err != nil {
		return "", err
	}

	q := url.Values{
		"info_hash":  []string{string(infoHash[:])},
		"peer_id":    []string{connection.peerId[:]},
		"port":       []string{strconv.Itoa(PORT)},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"left":       []string{strconv.FormatUint(totalLength, 10)},
		"conpact":    []string{"1"},
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func buildAnnounceRequest(connID uint64, txID uint32, infoHash, peerID []byte, totalSize uint64) []byte {
	buf := make([]byte, 98)
	binary.BigEndian.PutUint64(buf[0:8], connID)
	binary.BigEndian.PutUint32(buf[8:12], 1) // action = announce
	binary.BigEndian.PutUint32(buf[12:16], txID)

	copy(buf[16:36], infoHash)
	copy(buf[36:56], peerID)

	// downloaded, left, uploaded
	binary.BigEndian.PutUint64(buf[56:64], 0)
	binary.BigEndian.PutUint64(buf[64:72], totalSize)
	binary.BigEndian.PutUint64(buf[72:80], 0)

	binary.BigEndian.PutUint32(buf[80:84], 2)             // event = started
	binary.BigEndian.PutUint32(buf[84:88], 0)             // ip = default
	binary.BigEndian.PutUint32(buf[88:92], rand.Uint32()) // key
	binary.BigEndian.PutUint32(buf[92:96], 0xFFFFFFFF)            // num_want (50 peers)
	binary.BigEndian.PutUint16(buf[96:98], uint16(6881))  // port

	return buf
}

func makeConnectRequest(txID uint32) []byte {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[0:8], 0x41727101980) // protocol magic constant
	binary.BigEndian.PutUint32(buf[8:12], 0)            // action = connect
	binary.BigEndian.PutUint32(buf[12:16], txID)
	return buf
}
