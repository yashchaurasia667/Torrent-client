package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type File struct {
	length uint64
	path   []string
}

type Info struct {
	name        string
	pieceLength uint64
	pieces      []byte
	length      uint64
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

func getPieces(path string, pieceLength uint64) []byte {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Error openning file:", err)
		os.Exit(-1)
	}
	defer file.Close()

	var pieces []byte

	for {
		buffer := make([]byte, pieceLength)
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			fmt.Println("Error while reading the given file:", err)
			os.Exit(-1)
		}

		if n == 0 {
			break
		}

		hash := sha1.Sum(buffer[:n])
		pieces = append(pieces, hash[:]...)
	}

	return pieces
}

func traverseDirectory(path string) {
}

func getPath(pieceLength uint64) ([]File, []byte) {
	var path string
	var info os.FileInfo
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Path to the file [essential]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input: ", err)
			continue
		}
		path = strings.TrimSpace(input)
		if path == "" {
			continue
		}

		info, err = os.Stat(path)
		if os.IsNotExist(err) {
			fmt.Println("Path does not exist, Please enter a valid path.")
			continue
		} else if err != nil {
			fmt.Println("Error:", err)
			os.Exit(-1)
		}
		break
	}

	if info.IsDir() {
		fmt.Println("The given path is a directory.")
		return []File{}, []byte{}
	} else {
		fmt.Println("The given path is a file.")
		fmt.Printf("The size of the given file is: %d bytes \n", info.Size())

		delimeter := regexp.MustCompile(`[\\/|]+`)
		parts := delimeter.Split(path, -1)
		fmt.Println(parts[len(parts)-1])
		pieces := getPieces(path, pieceLength)

		return []File{
			{
				length: uint64(info.Size()),
				path:   parts,
			},
		}, pieces
	}
}

func getDetails() Torrent {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Couldn't get the username.")
		currentUser.Name = "User Not Found"
	}

	var meta = Torrent{
		announce:         "udp://tracker.openbittorrent.com:80/announce",
		announceList:     []string{},
		createdBy:        currentUser.Name,
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

	fmt.Printf("Created by [default: %s]: ", meta.createdBy)
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
		fmt.Println("Error reading input, using default:", err)
	}

	inp = strings.TrimSpace(inp)
	if inp != "" {
		uInp, err := strconv.Atoi(inp)
		if err != nil {
			fmt.Println("Invalid number, using default:", meta.info.pieceLength)
		} else {
			meta.info.pieceLength = uint64(uInp)
		}
	}

	info, pieces := getPath(meta.info.pieceLength * 1000000)
	switch len(info) {
	case 1:
		meta.info.length = info[0].length
		meta.info.pieces = pieces
	default:
		break
	}

	return meta
}

func main() {
	getDetails()
}
