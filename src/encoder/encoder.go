package main

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"regexp"
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

func getPath() []File {
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
		return []File{}
	} else {
		fmt.Println("The given path is a file.")
		fmt.Printf("The size of the given file is: %d bytes \n", info.Size())

		delimeter := regexp.MustCompile(`[\\/|]+`)
		parts := delimeter.Split(path, -1)
		fmt.Println(parts[len(parts)-1])

		return []File{
			{
				length: info.Size(),
				path:   strings.Split(path, "/"),
			},
		}
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
			meta.info.pieceLength = uInp
		}
	}

	res := getPath()
	switch len(res) {
	case 1:
		meta.info.length = res[0].length
	default:
		break
	}

	return meta
}

func main() {
	getDetails()
}
