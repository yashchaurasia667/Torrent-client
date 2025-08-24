package main

import (
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

type Connection struct {
	connectionType string
	connectionId   string
	client         http.Client
	peerId         string
}

var connection Connection

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

func BuildTrackerUrl(trakerAddr string, infoHash []byte, totalLength int64) (string, error) {
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

func UdpRequest(url string) {
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

	data := []byte("hello traker")
	_, err = conn.Write(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing to UDP connection: ", err)
		os.Exit(1)
	}

	fmt.Println("Sent UDP packet to", addr.String())
}

func HTTPRequest(url string) ([]byte, error) {
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

func RequestTracker(path string) {
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

	connection.peerId = GeneratePeerId()
	// rawHash, err := hex.DecodeString(t.InfoHash)
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, "Invalid infohash:", err)
	// 	os.Exit(1)
	// }

	if strings.HasPrefix(u.Scheme, "http") {
		connection.connectionType = "http"

		trackerAddr, err := BuildTrackerUrl(t.Announce, t.InfoHash, t.TotalLength)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error building tracker url: ", err)
			os.Exit(1)
		}
		body, err := HTTPRequest(trackerAddr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error while making Request to the tracker: ", err)
			os.Exit(1)
		}

		fmt.Println("Response body:", string(body))
	} else if u.Scheme == "udp" {
		connection.connectionType = "udp"
		fmt.Println("This is a UDP tracker using the UDP Request method")
	} else {
		connection.connectionType = "unknown"
		fmt.Println("This is an unknown protocol", u.Scheme)
	}

}

func main() {
	// UdpRequest("udp://tracker.opentrackr.org:1337/announce")
	RequestTracker("../test_files/debian-installer.torrent")
	// UdpRequest("http://bttracker.debian.org:6969/announce")
}
