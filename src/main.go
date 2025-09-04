package main

import (
	"fmt"
	"os"
	"strings"
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

	res, err := peers.RequestTracker(t)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = peers.StartPeerConnections(res.Peers, t.InfoHash, []byte(peers.GetPeerId()))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
