package main

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

// var connection Connection

// func urlEncode(b []byte) string {
// 	return url.QueryEscape(string(b))
// }

// func percentEncode(b []byte) string {
// 	var sb strings.Builder
// 	for _, c := range b {
// 		if (c >= 'a' && c <= 'z') ||
// 			(c >= 'A' && c <= 'Z') ||
// 			(c >= '0' && c <= '9') ||
// 			c == '-' || c == '_' || c == '.' || c == '~' {
// 			sb.WriteByte(c)
// 		} else {
// 			sb.WriteString(fmt.Sprintf("%%%02X", c))
// 		}
// 	}
// 	return sb.String()
// }

func GeneratePeerId() string {
	prefix := "-" + INIT + VERSION
	b := make([]byte, 13)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return prefix + string(b)
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

func UdpRequest(url string) (*ConnectionResponse, error) {
	// Remove the udp:// part
	u := strings.Split(url, "://")[1]

	// Seperate the host:port part from the complete url
	trakerAddr := strings.Split(u, "/")[0]
	// fmt.Println(trakerAddr)

	addr, err := net.ResolveUDPAddr("udp", trakerAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error resolving UDP address: ", err)
		os.Exit(1)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error Dialing UDP address: ", err)
		os.Exit(1)
	}
	defer conn.Close()

	var transactionId uint32
	binary.Read(crand.Reader, binary.BigEndian, &transactionId)

	req := ConnectionRequest{
		ProtocolId:    0x41727101980,
		Action:        0,
		TransactionId: transactionId,
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, req); err != nil {
		return nil, err
	}

	if _, err := conn.Write(buf.Bytes()); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing to UDP connection: ", err)
		return nil, err
	}
	fmt.Println("Sent connect Request...")

	respBuf := make([]byte, 16)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Read(respBuf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read response buffer...")
		return nil, err
	}

	var resp ConnectionResponse
	respReader := bytes.NewReader(respBuf)
	if err := binary.Read(respReader, binary.BigEndian, &resp); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse response...")
		return nil, err
	}

	if resp.TransactionId != transactionId {
		return nil, errors.New("transaction id mismatch")
	}
	if resp.Action != 0 {
		return nil, errors.New("invalid action response")
	}

	fmt.Println("Received Connection id ", resp.ConnectionId)
	return nil, nil
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
	fmt.Println("Successfully made a GET request to ", url)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func RequestTracker(path string) (*parser.Response, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read: ", err)
		os.Exit(1)
	}

	t, err := parser.AssembleTorrent(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error while reading torrent file: ", err)
		os.Exit(1)
	}

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
		res, err := r.DecodeResponse()
		if err != nil {
			return nil, err
		}
		return res, nil
	} else if u.Scheme == "udp" {
		fmt.Println("This is a UDP tracker using the UDP Request method")
		UdpRequest(t.Announce)
		return nil, fmt.Errorf("udp udp")
	} else {
		fmt.Println("This is an unknown protocol", u.Scheme)
		return nil, nil
	}

}

func main() {
	// UdpRequest("udp://tracker.opentrackr.org:1337/announce")
	res, err := RequestTracker("../test_files/single_file.torrent")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println(*res)
	// UdpRequest("http://bttracker.debian.org:6969/announce")

	// DEBUG
	// meta, err := parser.Test("../test_files/debian-installer.torrent")
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }

	// fmt.Println(meta.Announce)
}
