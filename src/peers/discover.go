package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
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

var peerId string

func urlEncode(b []byte) string {
	return url.QueryEscape(string(b))
}

func GeneratePeerId() string {
	prefix := "-" + INIT + VERSION
	b := make([]byte, 13)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return prefix + string(b)
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

	if strings.HasPrefix(u.Scheme, "http") {

		peerId = GeneratePeerId()
		// fmt.Println("peer id len ", len(peerId))

		rawHash, err := hex.DecodeString(t.InfoHash)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid infohash:", err)
			os.Exit(1)
		}

		q := u.Query()
		q.Add("info_hash", urlEncode(rawHash))
		q.Add("peer_id", urlEncode([]byte(peerId)))
		q.Add("port", fmt.Sprintf("%d", PORT))
		q.Add("uploaded", "0")
		q.Add("downloaded", "0")
		q.Add("left", fmt.Sprintf("%d", t.TotalLength))
		q.Add("compact", "1")
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
}
