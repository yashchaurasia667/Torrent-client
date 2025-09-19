package main

import (
	"fmt"
	"os"
	"strings"
	"torrent-client/download"
	"torrent-client/parser"
	"torrent-client/peers"
)

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

func main() {
	args := os.Args
	var downloaded []byte
	// var available_peers []*peers.ConnectedPeer

	// Exit if no file path is passed
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ./torrent-client [file path]")
		os.Exit(1)
	}
	// check for file and path validity
	check(args[1])
	// [DEBUG]
	// fmt.Println("ALL CHECKS PASSED...")

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

	for _, peer := range res.Peers {
		c, err := peers.PerformHandshake(peer, t.InfoHash, []byte(peers.GetPeerId()))
		if err != nil || c == nil {
			// fmt.Println("Error: ", err)
			continue
		}

		// available_peers = append(available_peers, c)
		intr := peers.CheckInterested(c.Conn)
		if intr {
			fmt.Println(peer.Ip.String(), "has unchoked you. Now requesting a piece")
			index, err := download.GetNextDownloadablePiece(c.Bitfield, downloaded)
			if err != nil {
				fmt.Println("Error: ", err)
				continue
			}

			// fmt.Println("Next downloadable index: ", index)
			peers.RequestPiece()
		}
	}
}
