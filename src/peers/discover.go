package peers

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"torrent-client/src/parser"
)

const RETRY_ATTEMPTS = 3

var connection HttpConnection

func UdpRequest(url string, infoHash []byte, peerId []byte, totalSize uint64) ([]byte, *AnnounceResponse, error) {
	trackerAddr := strings.Split(strings.Split(url, "://")[1], "/")[0]
	addr, err := net.ResolveUDPAddr("udp", trackerAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve udp: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, nil, fmt.Errorf("dial udp: %w", err)
	}
	defer conn.Close()

	// ---- CONNECT ----
	txID := generateTransactionId()
	connReq := makeConnectRequest(txID)

	respBuf := make([]byte, 16)
	var n int
	success := false
	for i := range RETRY_ATTEMPTS {
		_, err = conn.Write(connReq)
		if err != nil {
			return nil, nil, fmt.Errorf("write connect: %w", err)
		}

		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err = conn.Read(respBuf)
		if err == nil && n >= 16 {
			success = true
			break
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	if !success {
		return nil, nil, fmt.Errorf("no connect response after retries: %w", err)
	}

	action := binary.BigEndian.Uint32(respBuf[0:4])
	rxTxID := binary.BigEndian.Uint32(respBuf[4:8])
	connID := binary.BigEndian.Uint64(respBuf[8:16])

	if rxTxID != txID || action != 0 {
		return nil, nil, fmt.Errorf("bad connect response (action=%d, tx=%d)", action, rxTxID)
	}

	// ---- ANNOUNCE ----
	annTxID := generateTransactionId()
	annReq := buildAnnounceRequest(connID, annTxID, infoHash, peerId, totalSize)

	_, err = conn.Write(annReq)
	if err != nil {
		return nil, nil, fmt.Errorf("write announce: %w", err)
	}

	annResBuf := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	n, err = conn.Read(annResBuf)
	if err != nil {
		return nil, nil, fmt.Errorf("read announce: %w", err)
	}
	if n < 8 {
		return nil, nil, fmt.Errorf("announce too short, got %d bytes", n)
	}

	act := binary.BigEndian.Uint32(annResBuf[0:4])
	rxAnnTx := binary.BigEndian.Uint32(annResBuf[4:8])
	if act == 3 {
		return nil, nil, fmt.Errorf("tracker error: %s", string(annResBuf[8:n]))
	}
	if act != 1 || rxAnnTx != annTxID {
		return nil, nil, fmt.Errorf("bad announce response (action=%d, tx=%d)", act, rxAnnTx)
	}
	if n < 20 {
		fmt.Printf("Raw announce (%d bytes): %x\n", n, annResBuf[:n])
		return nil, nil, fmt.Errorf("announce response too short: got %d bytes", n)
	}

	interval := binary.BigEndian.Uint32(annResBuf[8:12])
	leechers := binary.BigEndian.Uint32(annResBuf[12:16])
	seeders := binary.BigEndian.Uint32(annResBuf[16:20])

	annRes := &AnnounceResponse{
		Action:        act,
		TransactionId: rxAnnTx,
		Interval:      interval,
		Leechers:      leechers,
		Seeders:       seeders,
	}

	peersData := annResBuf[20:n]
	for i := 0; i+6 <= len(peersData); i += 6 {
		ip := net.IPv4(peersData[i], peersData[i+1], peersData[i+2], peersData[i+3])
		port := binary.BigEndian.Uint16(peersData[i+4 : i+6])
		annRes.Peers = append(annRes.Peers, parser.Peer{Ip: ip, Port: port})
	}

	return annResBuf[:n], annRes, nil
}

func HTTPRequest(rawUrl string, connection *HttpConnection) ([]byte, error) {
	parsed, err := url.Parse(rawUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid tracker URL: %w", err)
	}

	host, port, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		host = parsed.Host
		port = "80"
	}

	addrs, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed %w", err)
	}

	var ipv4 string
	for _, addr := range addrs {
		ipv4 = addr.String()
		break
	}
	if ipv4 == "" {
		return nil, fmt.Errorf("no IPV4 address found for %s", host)
	}
	targetUrl := *parsed
	targetUrl.Host = net.JoinHostPort(ipv4, port)

	connection.client = http.Client{Timeout: 15 * time.Second}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		res, err := connection.client.Get(targetUrl.String())
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed: %w", attempt, err)
			fmt.Fprintf(os.Stderr, "HTTP tracker request failed (attempt %d): %v\n", attempt, err)
			time.Sleep(time.Second * time.Duration(attempt)) // simple backoff
			continue
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("non-200 status: %d %s", res.StatusCode, res.Status)
			fmt.Fprintf(os.Stderr, "Tracker responded with %s (attempt %d)\n", res.Status, attempt)
			time.Sleep(time.Second * time.Duration(attempt))
			continue
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			lastErr = fmt.Errorf("read body failed: %w", err)
			fmt.Fprintf(os.Stderr, "Failed to read tracker response: %v\n", err)
			time.Sleep(time.Second * time.Duration(attempt))
			continue
		}
		return body, nil
	}
	return nil, fmt.Errorf("tracker request failed after 3 attempts: %w", lastErr)
}

func RequestTracker(t *parser.Torrent, announce string) (*parser.Response, error) {
	u, err := url.Parse(announce)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error while parsing URL: ", err)
		os.Exit(1)
	}

	connection.peerId = generatePeerId()
	trackerAddr, err := buildTrackerUrl(announce, t.InfoHash, t.TotalLength, &connection)
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
		rawRes, annRes, err := UdpRequest(announce, t.InfoHash[:], []byte(connection.peerId), uint64(t.TotalLength))
		if err != nil {
			return nil, err
		}
		// fmt.Println("udp body: ", string(peers[:20]))
		peers, err := parser.DecodeUDPResponse(rawRes[20:])
		if err != nil {
			return nil, err
		}

		res := parser.Response{
			Interval: annRes.Interval,
			Peers:    peers[:],
		}
		return &res, nil
	} else {
		return nil, fmt.Errorf("this is an unknown protocol %s", u.Scheme)
	}

}

func Test() {
	// UdpRequest("udp://tracker.opentrackr.org:1337/announce")
	data, err := os.ReadFile("../test_files/single_file.torrent")
	if err != nil {
		fmt.Fprintln(os.Stderr, "read: ", err)
		os.Exit(1)
	}

	t, err := parser.DecodeTorrent(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error while reading torrent file: ", err)
		os.Exit(1)
	}

	res, err := RequestTracker(t, t.Announce)
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