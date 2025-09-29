package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"torrent-client/download"
	"torrent-client/parser"
	"torrent-client/peers"
)

const CONCURRENT_DONWLOADS = 10

type DownloadingSet struct {
	mu sync.RWMutex
	m  map[uint32]struct{}
}

type DownloadResult struct {
	pieceIndex uint32
	piece      []byte
	Err        error
}

func newDownloadingSet() *DownloadingSet {
	return &DownloadingSet{m: make(map[uint32]struct{})}
}

func (s *DownloadingSet) add(idx uint32) {
	s.mu.Lock()
	s.m[idx] = struct{}{}
	s.mu.Unlock()
}

func (s *DownloadingSet) remove(idx uint32) {
	s.mu.Lock()
	delete(s.m, idx)
	s.mu.Unlock()
}

func (s *DownloadingSet) contains(idx uint32) bool {
	s.mu.Lock()
	_, ok := s.m[idx]
	s.mu.Unlock()
	return ok
}

func check(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Invalid file path.")
		os.Exit(1)
	}
	parts := strings.Split(path, ".")
	if parts[len(parts)-1] != "torrent" {
		fmt.Fprintln(os.Stderr, "The file passed is not a torrent file.")
		os.Exit(1)
	}
}

func getNextPieceIndex(downloaded []byte, bitField []byte, downloading *DownloadingSet) (int, int, uint32, error) {
	for {
		dIndex, bIndex, err := download.GetNextDownloadablePiece(bitField, downloaded)
		if err != nil {
			return 0, 0, 0, err
		}
		pieceIndex := uint32(dIndex*8 + bIndex)

		if downloading.contains(pieceIndex) {
			downloaded[dIndex] += byte(1 << (7 - bIndex))
			continue
		}

		return dIndex, bIndex, pieceIndex, nil
	}
}

/*
This function will be called asynchrousnously

Args=>
peer: struct containing info about peer that the tracker sent (IP, Port)
t: struct containing parsed torrent information
downloaded: slice containing info about all the pieces that have been downloaded
downloading: channel containing info about all the pieces that are currently downloading
wg: waitgroup to create a joining point to the main function
*/

func GetPeerAndDownload(peer parser.Peer, t *parser.Torrent, downloaded []byte, peerId []byte, downloading DownloadingSet, wg *sync.WaitGroup) ([]DownloadResult, error) {
	var pieces []DownloadResult
	c, err := peers.PerformHandshake(peer, t.InfoHash, peerId)
	if err != nil || c == nil {
		return nil, err
	}
	defer c.Conn.Close()
	defer wg.Done()

	intr := peers.CheckInterested(c.Conn)
	if !intr {
		c.Conn.Close()
		return nil, fmt.Errorf("%s is not interested", peer.Ip.String())
	}
	fmt.Printf("%s has unchoked you. Now requesting a piece\n", peer.Ip.String())

	// download all the available pieces that peer offers
	for {
		tmp := append([]byte(nil), downloaded...)
		dIndex, bIndex, pieceIndex, err := getNextPieceIndex(tmp, c.Bitfield, &downloading)
		if err != nil {
			return nil, err
		}
		downloaded[dIndex] += byte(1 << (7 - bIndex))
		downloading.add(pieceIndex)

		piece, err := download.DownloadPiece(c.Conn, c.Bitfield, t, pieceIndex)
		if err != nil {
			downloaded[dIndex] -= byte(1 << (7 - bIndex))
			break
		}
		fmt.Printf("Downloaded piece index %d from peer %s\n", pieceIndex, peer.Ip.String())
		pieces = append(pieces, DownloadResult{pieceIndex, piece, err})
		downloading.remove(pieceIndex)

		pathList := [2]string{"..", "out"}
		err = download.WritePiece(pieceIndex, piece, pathList[:])
		if err != nil {
			return nil, err
		}
	}
	return pieces, nil
}

// TODO: create either a central scheduler or a mutex for locking

func main() {
	args := os.Args
	var downloaded []byte
	var wg sync.WaitGroup
	downloading := newDownloadingSet()

	// Exit if no file path is passed
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ./torrent-client [file path]")
		os.Exit(1)
	}
	// check for file and path validity
	check(args[1])

	// Read file and Get decoded struct
	data, err := os.ReadFile(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Read: ", err)
		os.Exit(1)
	}

	t, err := parser.AssembleTorrent(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error while reading torrent file: ", err)
		os.Exit(1)
	}
	downloaded = make([]byte, t.Info.PieceCount/8)
	res, err := peers.RequestTracker(t)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	peerId := peers.GetPeerId()

	fmt.Printf("Piece Length: %d, block size: %d\n", t.Info.PieceLength, download.BLOCK_SIZE)
	for _, peer := range res.Peers {
		wg.Add(1)
		go func(p parser.Peer) {
			_, err := GetPeerAndDownload(p, t, downloaded, []byte(peerId), downloading, &wg)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				return
			}
		}(peer)
	}
	wg.Wait()
}
