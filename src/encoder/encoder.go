package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
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
	name             string
	announce         string
	announceList     []string
	createdBy        string
	creationDate     int64
	encoding         string
	comment          string
	hasMultipleFiles bool
	info             Info
}

// --- Bencode Helpers ---

func bencodeString(value string) string {
	return fmt.Sprintf("%d:%s", len(value), value)
}

func bencodeInt(value uint64) string {
	return fmt.Sprintf("i%de", value)
}

func bencodeBytes(b []byte) string {
	return fmt.Sprintf("%d:%s", len(b), string(b))
}

func bencodeFileList(files []File) string {
	var out strings.Builder
	out.WriteString("l")
	for _, f := range files {
		out.WriteString("d")
		out.WriteString(bencodeString("length"))
		out.WriteString(bencodeInt(f.length))

		out.WriteString(bencodeString("path"))
		out.WriteString("l")
		for _, part := range f.path {
			out.WriteString(bencodeString(part))
		}
		out.WriteString("e") // end path list
		out.WriteString("e") // end file dict
	}
	out.WriteString("e") // end file list
	return out.String()
}

func bencodeInfo(info Info, hasMultipleFiles bool) string {
	var out strings.Builder
	out.WriteString("d")

	// Sorted keys
	out.WriteString(bencodeString("name"))
	out.WriteString(bencodeString(info.name))

	out.WriteString(bencodeString("piece length"))
	out.WriteString(bencodeInt(info.pieceLength))

	if hasMultipleFiles {
		out.WriteString(bencodeString("files"))
		out.WriteString(bencodeFileList(info.files))
	} else {
		out.WriteString(bencodeString("length"))
		out.WriteString(bencodeInt(info.length))
	}

	out.WriteString(bencodeString("pieces"))
	out.WriteString(bencodeBytes(info.pieces))

	out.WriteString("e") // end info dict
	return out.String()
}

// --- Torrent Helpers ---

func getSHA1Sum(piece []byte) []byte {
	hash := sha1.Sum(piece)
	return hash[:]
}

func createPath(path string) []string {
	delimeter := regexp.MustCompile(`[\\/|]+`)
	return delimeter.Split(path, -1)
}

func encryptFiles(files []File, pieceLength uint64) []byte {
	var pieces []byte
	var buffer []byte

	for _, file := range files {
		path := strings.Join(file.path, "/")
		f, err := os.Open(path)
		if err != nil {
			fmt.Printf("Failed to open file [%s]: %e\n", path, err)
			os.Exit(-1)
		}
		defer f.Close()

		tmp := make([]byte, file.length)
		_, err = f.Read(tmp)
		if err != nil && err != io.EOF {
			fmt.Printf("Error reading file [%s]: %e\n", path, err)
			os.Exit(-1)
		}

		buffer = append(buffer, tmp...)
	}

	for i := uint64(0); i+pieceLength <= uint64(len(buffer)); i += pieceLength {
		piece := getSHA1Sum(buffer[i : i+pieceLength])
		pieces = append(pieces, piece...)
	}

	// Handle final piece (if not aligned)
	if rem := uint64(len(buffer)) % pieceLength; rem != 0 {
		start := uint64(len(buffer)) - rem
		piece := getSHA1Sum(buffer[start:])
		pieces = append(pieces, piece...)
	}

	return pieces
}

func traverseDirectory(path string) []File {
	var paths []File
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			fmt.Printf("Failed to stat [%s]: %e\n", path, err)
			os.Exit(-1)
		}
		if !info.IsDir() {
			paths = append(paths, File{uint64(info.Size()), createPath(path)})
		}
		return nil
	})
	if err != nil {
		fmt.Println("Directory walk failed:", err)
	}
	return paths
}

func getPath(pieceLength uint64) ([]File, []byte) {
	var path string
	var info os.FileInfo
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Path to the file or directory [required]: ")
		input, _ := reader.ReadString('\n')
		path = strings.TrimSpace(input)
		if path == "" {
			continue
		}

		var err error
		info, err = os.Stat(path)
		if os.IsNotExist(err) {
			fmt.Println("Path does not exist.")
			continue
		} else if err != nil {
			fmt.Println("Error:", err)
			os.Exit(-1)
		}
		break
	}

	if info.IsDir() {
		files := traverseDirectory(path)
		pieces := encryptFiles(files, pieceLength)
		return files, pieces
	} else {
		file := []File{{uint64(info.Size()), createPath(path)}}
		pieces := encryptFiles(file, pieceLength)
		return file, pieces
	}
}

func getDetails() Torrent {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Couldn't get the username.")
		currentUser.Name = "User Not Found"
	}

	var meta = Torrent{
		name:             "",
		announce:         "udp://tracker.openbittorrent.com:80/announce",
		announceList:     []string{},
		createdBy:        currentUser.Name,
		creationDate:     time.Now().Unix(),
		encoding:         "UTF-8",
		comment:          "",
		hasMultipleFiles: false,
		info: Info{
			name:        "",
			pieceLength: 16 * 1024,
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
		meta.announceList = append(meta.announceList, input)
	}

	fmt.Printf("Created by [default: %s]: ", meta.createdBy)
	meta.createdBy, _ = reader.ReadString('\n')
	meta.createdBy = strings.TrimSpace(meta.createdBy)

	fmt.Print("Comment [default: \"\"]: ")
	meta.comment, _ = reader.ReadString('\n')
	meta.comment = strings.TrimSpace(meta.comment)

	for {
		fmt.Print("Name of the file [required]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input: ", err)
			continue
		}
		meta.name = strings.TrimSpace(input)

		if meta.name != "" {
			break
		}
	}

	fmt.Printf("Piece Size [default: %dKB]: ", meta.info.pieceLength/1024)
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
			meta.info.pieceLength = uint64(uInp) * 1024
		}
	}

	files, pieces := getPath(meta.info.pieceLength)
	switch len(files) {
	case 1:
		meta.info.length = files[0].length
		meta.info.pieces = pieces
		meta.info.name = files[0].path[len(files[0].path)-1]
	default:
		meta.hasMultipleFiles = true
		meta.info.name = meta.name
		meta.info.files = files
		meta.info.pieces = pieces
	}

	return meta
}

func createTorrent() {
	meta := getDetails()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Output path [default: ./]: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		input = "./"
	}

	var out strings.Builder
	out.WriteString("d")
	out.WriteString(bencodeString("announce"))
	out.WriteString(bencodeString(meta.announce))

	if len(meta.announceList) > 0 {
		out.WriteString(bencodeString("announce-list"))
		out.WriteString("l")
		for _, tracker := range meta.announceList {
			out.WriteString("l")
			out.WriteString(bencodeString(tracker))
			out.WriteString("e")
		}
		out.WriteString("e")
	}

	out.WriteString(bencodeString("created by"))
	out.WriteString(bencodeString(meta.createdBy))

	out.WriteString(bencodeString("creation date"))
	out.WriteString(bencodeInt(uint64(meta.creationDate)))

	out.WriteString(bencodeString("encoding"))
	out.WriteString(bencodeString(meta.encoding))

	if meta.comment != "" {
		out.WriteString(bencodeString("comment"))
		out.WriteString(bencodeString(meta.comment))
	}

	out.WriteString(bencodeString("info"))
	out.WriteString(bencodeInfo(meta.info, meta.hasMultipleFiles))
	out.WriteString("e") // end root dict

	outputPath := filepath.Join(input, meta.name+".torrent")
	file, err := os.Create(outputPath)
	if err != nil {
		fmt.Println("Failed to create torrent file:", err)
		os.Exit(-1)
	}
	defer file.Close()

	_, err = file.WriteString(out.String())
	if err != nil {
		fmt.Println("Failed to write torrent file:", err)
		os.Exit(-1)
	}

	fmt.Println("Torrent created at:", outputPath)
}

func main() {
	createTorrent()
}
