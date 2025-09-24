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

type DownloadResult struct {
	pieceIndex uint32
	piece      []byte
	Err        error
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

func GetPeerAndDownload(peer parser.Peer, t *parser.Torrent, downloaded []download.DownloadedBit, peerId []byte, wg *sync.WaitGroup) ([]DownloadResult, error) {
	var pieces []DownloadResult
	c, err := peers.PerformHandshake(peer, t.InfoHash, peerId)
	if err != nil || c == nil {
		return nil, err
	}
	defer c.Conn.Close()

	intr := peers.CheckInterested(c.Conn)
	if !intr {
		c.Conn.Close()
		return nil, fmt.Errorf("%s is not interested", peer.Ip.String())
	}

	fmt.Printf("%s has unchoked you. Now requesting a piece\n", peer.Ip.String())
	for {
		pieceIndex, piece, err := download.DownloadPiece(c.Conn, c.Bitfield, downloaded, t)
		if err != nil {
			// return nil, err
			break
		}
		pieces = append(pieces, DownloadResult{pieceIndex, piece, err})

		fmt.Println("Downloaded a piece, piece length: ", len(piece))
	}
	wg.Done()
	return pieces, nil
}

func main() {
	args := os.Args
	var wg sync.WaitGroup
	var downloaded []download.DownloadedBit
	downloadResult := make(chan DownloadResult)

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
	downloaded = make([]download.DownloadedBit, t.Info.PieceCount/8)

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
			downloadedPieces, err := GetPeerAndDownload(p, t, downloaded, []byte(peerId), &wg)
			if err != nil || downloadedPieces == nil {
				fmt.Printf("Error: %s\n", err)
				return
			}
			for _, piece := range downloadedPieces {
				downloadResult <- piece
			}
		}(peer)
	}
	wg.Wait()
}
