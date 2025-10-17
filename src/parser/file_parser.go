package parser

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"time"
)

type Torrent struct {
	Announce         string
	AnnounceList     []string
	CreatedBy        string
	CreationDate     *time.Time
	Comment          string
	Encoding         string
	Info             InfoDict
	InfoHash         []byte
	HasMultipleFiles bool
	TotalLength      uint64
	Magnet           string
}

type InfoDict struct {
	Name        string
	Length      uint64
	PieceLength uint64
	Pieces      []byte
	PieceHashes [][]byte
	PieceCount  uint32
	Private     bool
	Files       []InfoFile
}

type InfoFile struct {
	Length uint64
	Path   []string
}

type Reader struct {
	b   []byte
	pos int
}

// Read implements io.Reader.
func (r *Reader) Read(p []byte) (n int, err error) {
	panic("unimplemented")
}

func NewReader(b []byte) *Reader {
	return &Reader{b: b}
}

// READER UTIL FUNCTION
func (r *Reader) readByte() (byte, error) {
	if r.pos >= len(r.b) {
		return 0, io.ErrUnexpectedEOF
	}
	r.pos++
	return r.b[r.pos-1], nil
}

func (r *Reader) expectByte(b byte) error {
	ch, err := r.readByte()
	if err != nil {
		return err
	}

	if ch != b {
		return fmt.Errorf("bencode expected %c got %c at %d", b, ch, r.pos-1)
	}
	return nil
}

func (r *Reader) peek() (byte, error) {
	if r.pos > len(r.b) {
		return 0, io.ErrUnexpectedEOF
	}
	return r.b[r.pos], nil
}

func (r *Reader) readString() (string, error) {
	len := 0

	for {
		ch, err := r.readByte()
		if err != nil {
			return "", err
		}

		n := ch - '0'
		if n <= 9 {
			len = len*10 + int(n)
		} else if ch == ':' {
			break
		} else {
			return "", fmt.Errorf("invalid formatting for bencoded string")
		}
	}

	r.pos += len
	return string(r.b[r.pos-len : r.pos]), nil
}

func (r *Reader) readStringList() ([]string, error) {
	err := r.expectByte('l')
	if err != nil {
		return nil, err
	}

	elems := []string{}

	for {
		ch, err := r.peek()
		if err != nil {
			return nil, err
		}

		if ch == 'l' {
			s, err := r.readStringList()
			if err != nil {
				return nil, err
			}
			elems = append(elems, s...)
		} else if ch <= '9' {
			s, err := r.readString()
			if err != nil {
				return nil, err
			}
			elems = append(elems, s)
		} else if ch == 'e' {
			r.readByte()
			break
		}
	}

	return elems, nil
}

func (r *Reader) readInt() (uint64, error) {
	err := r.expectByte('i')
	if err != nil {
		return 0, err
	}

	var num uint64 = 0
	for {
		ch, err := r.readByte()
		if err != nil {
			return 0, err
		}

		if ch == 'e' {
			break
		}

		// num = num*10 + uint64(ch-'0')
		num = num*10 + uint64(ch-'0')
	}

	return num, nil
}

func (r *Reader) skipAny() error {
	ch, err := r.peek()
	if err != nil {
		return err
	}

	switch ch {
	case 'i':
		r.readInt()
	case 'l':
		r.readByte()
		for {
			ch, err := r.peek()
			if err != nil {
				return err
			}
			if ch == 'e' {
				r.readByte()
				return nil
			}
			if err := r.skipAny(); err != nil {
				return err
			}
		}

	case 'd':
		r.readByte()
		for {
			ch, err := r.peek()
			if err != nil {
				return err
			}
			if ch == 'e' {
				r.readByte()
				return nil
			}
			// key
			if _, err := r.readString(); err != nil {
				return err
			}
			// value
			if err := r.skipAny(); err != nil {
				return err
			}
		}
	default:
		_, err := r.readString()
		return err
	}

	return nil
}

func (r *Reader) readInfoFile() (*InfoFile, error) {
	err := r.expectByte('d')
	if err != nil {
		return nil, err
	}

	var file InfoFile
	for {
		ch, err := r.peek()
		if err != nil {
			return nil, err
		}

		if ch == 'e' {
			r.readByte()
			break
		}

		key, err := r.readString()
		if err != nil {
			return nil, err
		}

		switch key {
		case "length":
			i, err := r.readInt()
			if err != nil {
				return nil, err
			}
			file.Length = i

		case "path":
			elems, err := r.readStringList()
			if err != nil {
				return nil, err
			}
			file.Path = elems[:]

		default:
			r.skipAny()
		}
	}

	return &file, err
}

func (r *Reader) readInfoFilesList() ([]InfoFile, error) {
	err := r.expectByte('l')
	if err != nil {
		return nil, err
	}

	var files []InfoFile
	for {
		ch, err := r.peek()
		if err != nil {
			return nil, err
		}

		if ch == 'e' {
			r.readByte()
			break
		}

		f, err := r.readInfoFile()
		if err != nil {
			return nil, err
		}
		files = append(files, *f)
	}
	return files, nil
}

func GetSha1Hash(b []byte) []byte {
	sha := sha1.New()
	sha.Write(b)
	rawHash := sha.Sum(nil)
	return rawHash
}

func (r *Reader) readInfo() (*InfoDict, error) {
	err := r.expectByte('d')
	if err != nil {
		return nil, err
	}

	var info InfoDict

	for {
		ch, err := r.peek()
		if err != nil {
			return nil, err
		}

		if ch == 'e' {
			r.readByte()
			break
		}

		key, err := r.readString()
		if err != nil {
			return nil, err
		}

		switch key {
		case "name":
			s, err := r.readString()
			if err != nil {
				return nil, err
			}
			info.Name = s

		case "length":
			i, err := r.readInt()
			if err != nil {
				return nil, err
			}
			info.Length = i

		case "piece length":
			i, err := r.readInt()
			if err != nil {
				return nil, err
			}
			info.PieceLength = uint64(i)

		case "pieces":
			var piecesLen uint32 = 0
			for {
				ch, err := r.readByte()
				if err != nil {
					return nil, err
				}

				if ch == ':' {
					break
				}

				if ch > '9' {
					return nil, fmt.Errorf("invalid format for pieces in info dict")
				}

				piecesLen = piecesLen*10 + uint32(ch-'0')
			}
			info.Pieces = r.b[r.pos : r.pos+int(piecesLen)]
			r.pos += int(piecesLen)

			// CALCULATE PIECE HASHES
			info.PieceCount = piecesLen / 20

			pieceHashes := make([][]byte, info.PieceCount)
			for i := range info.PieceCount {
				start := i * 20
				end := start + 20
				pieceHashes[i] = info.Pieces[start:end]
			}
			info.PieceHashes = pieceHashes

		case "private":
			i, err := r.readInt()
			if err != nil {
				return nil, err
			}
			info.Private = (i != 0)

		case "files":
			files, err := r.readInfoFilesList()
			if err != nil {
				return nil, err
			}
			info.Files = files

		default:
			r.skipAny()
		}
	}

	return &info, nil
}

// TORRENT FUNCTIONS
func DecodeTorrent(data []byte) (*Torrent, error) {
	r := NewReader(data)
	err := r.expectByte('d')
	if err != nil {
		return nil, err
	}

	var meta Torrent

	for {
		ch, err := r.peek()
		if err != nil {
			return nil, err
		}

		if ch == 'e' {
			r.readByte()
			break
		}

		key, err := r.readString()
		if err != nil {
			return nil, err
		}

		switch key {
		case "announce":
			s, err := r.readString()
			if err != nil {
				return nil, err
			}
			meta.Announce = s

		case "announce-list":
			elems, err := r.readStringList()
			if err != nil {
				return nil, err
			}
			meta.AnnounceList = elems

		case "comment":
			s, err := r.readString()
			if err != nil {
				return nil, err
			}
			meta.Comment = s

		case "created by":
			s, err := r.readString()
			if err != nil {
				return nil, err
			}
			meta.CreatedBy = s

		case "creation date":
			i, err := r.readInt()
			if err != nil {
				return nil, err
			}
			t := time.Unix(int64(i), 0).UTC()
			meta.CreationDate = &t

		case "encoding":
			s, err := r.readString()
			if err != nil {
				return nil, err
			}
			meta.Encoding = s

		case "info":
			infoStart := r.pos
			info, err := r.readInfo()
			if err != nil {
				return nil, err
			}
			infoEnd := r.pos
			meta.Info = *info
			meta.InfoHash = GetSha1Hash(r.b[infoStart:infoEnd])

			if len(meta.Info.Files) > 0 {
				meta.HasMultipleFiles = true
			}

		default:
			r.skipAny()
		}
	}

	if meta.HasMultipleFiles {
		for _, i := range meta.Info.Files {
			// fmt.Printf("%s of size %d\n", i.Path[len(i.Path)-1], i.Length)
			meta.TotalLength += i.Length
		}
	} else {
		meta.TotalLength = uint64(meta.Info.Length)
	}

	return &meta, nil
}

func Test(path string) (*Torrent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return DecodeTorrent(data)
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, "Error while decoding torrent file: ", err)
	// 	os.Exit(1)
	// }
}
