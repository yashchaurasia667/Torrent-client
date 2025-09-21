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

func GetPeerAndDownload(peer parser.Peer, t *parser.Torrent, downloaded []byte) (uint32, []byte, error) {
	c, err := peers.PerformHandshake(peer, t.InfoHash, []byte(peers.GetPeerId()))
	if err != nil || c == nil {
		return 0, nil, err
	}

	intr := peers.CheckInterested(c.Conn)
	if !intr {
		return 0, nil, fmt.Errorf("%s is not interested", peer.Ip.String())
	}

	fmt.Println(peer.Ip.String(), "has unchoked you. Now requesting a piece")
	pieceIndex, piece, err := download.DownloadPiece(c.Conn, c.Bitfield, downloaded, t)
	if err != nil {
		return 0, nil, err
	}

	fmt.Println("Downloaded a piece, piece length: ", len(piece))
	return pieceIndex, piece, nil
}

func main() {
	args := os.Args
	var downloaded []byte

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

	fmt.Printf("Piece Length: %d, block size: %d\n", t.Info.PieceLength, download.BLOCK_SIZE)
	for _, peer := range res.Peers {
		_, piece, err := GetPeerAndDownload(peer, t, downloaded)
		if err != nil || piece == nil {
			fmt.Printf("Error: %s\n", err)
			continue
		}
	}
}
