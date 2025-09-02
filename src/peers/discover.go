package peers

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"torrent-client/parser"
)

const PORT = 6881
const INIT = "FS"
const VERSION = "0001"
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
	Inverval      uint32
	Leechers      uint32
	Seeders       uint32
}

func GeneratePeerId() string {
	prefix := "-" + INIT + VERSION
	b := make([]byte, 13)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return prefix + string(b)
}

func GenerateTransactionId() uint32 {
	var b [4]byte
	crand.Read(b[:])
	return binary.BigEndian.Uint32(b[:])
}

func BuildTrackerUrl(trakerAddr string, infoHash []byte, totalLength int64, connection *HttpConnection) (string, error) {
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
		"left":       []string{strconv.Itoa(int(totalLength))},
		"conpact":    []string{"1"},
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func UdpRequest(url string, infoHash []byte, peerId []byte, totalSize uint64) ([]byte, *AnnounceResponse, error) {
	// Remove the udp:// part
	u := strings.Split(url, "://")[1]

	// Seperate the host:port part from the complete url
	trakerAddr := strings.Split(u, "/")[0]
	addr, err := net.ResolveUDPAddr("udp", trakerAddr)
	if err != nil {
		return nil, nil, err
		// fmt.Errorf("Error resolving UDP address: ", err)
		// os.Exit(1)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		// fmt.Fprintln(os.Stderr, "Error Dialing UDP address: ", err)
		// os.Exit(1)
		return nil, nil, err
	}
	defer conn.Close()

	// Make connect request
	req := ConnectionRequest{
		ProtocolId:    0x41727101980,
		Action:        0,
		TransactionId: GenerateTransactionId(),
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, req); err != nil {
		return nil, nil, err
	}
	if _, err := conn.Write(buf.Bytes()); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing to UDP connection: ", err)
		return nil, nil, err
	}
	// fmt.Println("Sent connect Request...")

	// Receive connect response
	respBuf := make([]byte, 16)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Read(respBuf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read response buffer...")
		return nil, nil, err
	}
	var resp ConnectionResponse
	respReader := bytes.NewReader(respBuf)
	if err := binary.Read(respReader, binary.BigEndian, &resp); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse response...")
		return nil, nil, err
	}

	if resp.TransactionId != req.TransactionId {
		return nil, nil, errors.New("transaction id mismatch")
	}
	if resp.Action != 0 {
		return nil, nil, errors.New("invalid action response")
	}
	// fmt.Println("Received Connection id ", resp.ConnectionId)

	// Make announce request
	annReq := AnnounceRequest{
		ConnectionId:  resp.ConnectionId,
		Action:        1,
		TransactionId: GenerateTransactionId(),
		Downloaded:    0,
		Left:          totalSize,
		Uploaded:      0,
		Event:         2,
		IpAddress:     0,
		Key:           rand.Uint32(),
		NumWant:       NUM_PEERS,
		Port:          PORT,
	}
	copy(annReq.InfoHash[:], infoHash)
	copy(annReq.PeerId[:], peerId)

	buf.Reset()
	binary.Write(buf, binary.BigEndian, annReq)
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return nil, nil, err
	}
	// fmt.Println("Set Announce Request")
	// fmt.Println("Announce Request ", annReq)

	// Receive announce response
	annResBuf := make([]byte, 2048)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Read(annResBuf)
	if err != nil {
		// fmt.Println("error ", err)
		return nil, nil, err
	}

	var annRes AnnounceResponse
	respReader = bytes.NewReader(annResBuf[:20])
	binary.Read(respReader, binary.BigEndian, &annRes)

	if annRes.TransactionId != annReq.TransactionId {
		return nil, nil, fmt.Errorf("transaction id mismatch in announce response")
	}

	return annResBuf, &annRes, nil
}

func HTTPRequest(url string, connection *HttpConnection) ([]byte, error) {
	connection.client = http.Client{Timeout: 15 * time.Second}

	res, err := connection.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Received non 200 response: %d %s \n", res.StatusCode, res.Status)
		os.Exit(1)
	}

	// DEBUG
	// fmt.Println("Successfully made a GET request to ", url)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func RequestTracker(t *parser.Torrent) (*parser.Response, error) {
	u, err := url.Parse(t.Announce)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error while parsing URL: ", err)
		os.Exit(1)
	}

	var connection HttpConnection

	connection.peerId = GeneratePeerId()
	trackerAddr, err := BuildTrackerUrl(t.Announce, t.InfoHash, t.TotalLength, &connection)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error building tracker url: ", err)
		os.Exit(1)
	}

	if strings.HasPrefix(u.Scheme, "http") {
		body, err := HTTPRequest(trackerAddr, &connection)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error while making Request to the tracker: ", err)
			os.Exit(1)
		}

		r := parser.NewReader(body)
		res, err := r.DecodeHttpResponse()
		if err != nil {
			return nil, err
		}
		return res, nil
	} else if u.Scheme == "udp" {
		// fmt.Println("This is a UDP tracker using the UDP Request method")
		rawRes, annRes, err := UdpRequest(t.Announce, t.InfoHash[:], []byte(connection.peerId), uint64(t.TotalLength))
		if err != nil {
			return nil, err
		}
		// fmt.Println("udp body: ", string(peers[:20]))
		peers, err := parser.DecodeUDPResponse(rawRes[20:])
		if err != nil {
			return nil, err
		}

		res := parser.Response{
			Interval: annRes.Inverval,
			Peers:    peers[:NUM_PEERS],
		}
		return &res, nil
	} else {
		fmt.Println("This is an unknown protocol", u.Scheme)
		return nil, nil
	}

}

func Test() {
	// UdpRequest("udp://tracker.opentrackr.org:1337/announce")
	data, err := os.ReadFile("../test_files/single_file.torrent")
	if err != nil {
		fmt.Fprintln(os.Stderr, "read: ", err)
		os.Exit(1)
	}

	t, err := parser.AssembleTorrent(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error while reading torrent file: ", err)
		os.Exit(1)
	}

	res, err := RequestTracker(t)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println(res)

	// UdpRequest("http://bttracker.debian.org:6969/announce")

	// DEBUG
	// meta, err := parser.Test("../test_files/debian-installer.torrent")
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }

	// fmt.Println(meta.Announce)
}
