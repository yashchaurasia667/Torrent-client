package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"torrent-client/parser"
)

const PORT = 6881
const INIT = "FS"
const VERSION = "0001"
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type UDPConnection struct {
	connectionId string
	peerId       string
}

var connection UDPConnection

func urlEncode(b []byte) string {
	return url.QueryEscape(string(b))
}

func percentEncode(b []byte) string {
	var sb strings.Builder
	for _, c := range b {
		if (c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			sb.WriteByte(c)
		} else {
			sb.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return sb.String()
}

func GeneratePeerId() string {
	prefix := "-" + INIT + VERSION
	b := make([]byte, 13)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return prefix + string(b)
}

func UdpRequest(url string) {
	u := strings.Split(url, "://")[1]
	fmt.Println(u)
	addr, err := net.ResolveUDPAddr("udp", u)
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

	fmt.Println(t.Announce)
	u, err := url.Parse(t.Announce)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error while parsing URL: ", err)
		os.Exit(1)
	}

	if strings.HasPrefix(u.Scheme, "http") {

		connection.peerId = GeneratePeerId()
		// fmt.Println("peer id len ", len(connection.peerId))

		rawHash, err := hex.DecodeString(t.InfoHash)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid infohash:", err)
			os.Exit(1)
		}

		// fmt.Println("Raw info hash: ", len(t.InfoHash))

		q := u.Query()
		q.Add("info_hash", percentEncode(rawHash))
		q.Add("peer_id", percentEncode([]byte(connection.peerId)))
		q.Add("port", fmt.Sprintf("%d", PORT))
		q.Add("uploaded", "0")
		q.Add("downloaded", "0")
		q.Add("left", fmt.Sprintf("%d", t.TotalLength))
		// q.Add("compact", "1")
		u.RawQuery = q.Encode()

		res, err := http.Get(u.String())
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error making request to the Tracker: ", err)
			os.Exit(1)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "Received non 200 response: %d %s \n", res.StatusCode, res.Status)
			os.Exit(1)
		}

		// DEBUG
		fmt.Println("Successfully made a GET request to ", t.Announce)

		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading response body:", err)
			os.Exit(1)
		}
		fmt.Println("Response body:", string(body))
	} else if u.Scheme == "udp" {
		fmt.Println("This is a UDP tracker you need to implement UDP protocol.")
	} else {
		fmt.Println("This is an unknown protocol", u.Scheme)
	}

}

func main() {
	RequestTracker("../test_files/debian-installer.torrent")
	// UdpRequest("http://bttracker.debian.org:6969/announce")
}
