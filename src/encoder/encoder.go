package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type File struct {
	length int64
	path   []string
}

type Info struct {
	name        string
	pieceLength int
	pieces      []byte
	length      int64
	files       []File
}

type Torrent struct {
	announce         string
	announceList     []string
	createdBy        string
	creationDate     int64
	encoding         string
	comment          string
	hasMultipleFiles bool
	info             Info
}

var meta = Torrent{
	announce:         "udp://tracker.openbittorrent.com:80/announce",
	announceList:     []string{},
	createdBy:        "",
	creationDate:     time.Now().Unix(),
	encoding:         "UTF-8",
	comment:          "",
	hasMultipleFiles: false,
	info: Info{
		name:        "",
		pieceLength: 2,
		pieces:      []byte{},
		length:      0,
		files:       []File{},
	},
}

func getDetails() {
	var path string

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Announce URL [default: %s]: ", meta.announce)
	fmt.Scanln(&meta.announce)

	fmt.Println("Announce List [default: []]: ")
	for {
		var input string
		fmt.Scanln(&input)

		if input == "" {
			break
		}
	}

	fmt.Print("Author: ")
	meta.createdBy, _ = reader.ReadString('\n')
	meta.createdBy = strings.TrimSpace(meta.createdBy)

	fmt.Print("Comment [default: \"\"]: ")
	meta.comment, _ = reader.ReadString('\n')
	meta.comment = strings.TrimSpace(meta.comment)

	for {
		fmt.Print("Name of the file [essential]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input: ", err)
			continue
		}
		meta.info.name = strings.TrimSpace(input)

		if meta.info.name != "" {
			break
		}
	}

	fmt.Printf("Piece Size [default: %dMB]: ", meta.info.pieceLength)
	inp, err := reader.ReadString('\n')

	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	inp = strings.TrimSpace(inp)
	if inp != "" {
		uInp, err := strconv.Atoi(inp)
		if err != nil {
			fmt.Println("Invalid number, using default:", meta.info.pieceLength)
		} else {
			meta.info.pieceLength = uInp
		}
	}

	for {
		fmt.Print("Path to the file [essential]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input: ", err)
			continue
		}
		path = strings.TrimSpace(input)

		if path != "" {
			break
		}
	}

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Println("Path does not exist")
	} else if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if info.IsDir() {
		fmt.Println("The given path is a directory.")
	} else {
		fmt.Println("The given path is a file.")
	}

}

func main() {
	getDetails()
}
