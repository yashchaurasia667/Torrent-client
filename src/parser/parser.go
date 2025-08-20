package parser

import (
	"crypto/sha1"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// -----------------------------
// Public data structures
// -----------------------------

type Torrent struct {
	Announce     string        `json:"announce,omitempty"`
	AnnounceList [][]string    `json:"announceList,omitempty"`
	CreationDate *time.Time    `json:"creationDate,omitempty"`
	Comment      string        `json:"comment,omitempty"`
	CreatedBy    string        `json:"createdBy,omitempty"`
	Encoding     string        `json:"encoding,omitempty"`
	Info         InfoDict      `json:"info"`
	InfoHash     string        `json:"infoHashHex"` // hex, lower-case
	InfoHashB32  string        `json:"infoHashBase32"`
	PieceHashes  []string      `json:"pieceHashesHex"`
	TotalLength  int64         `json:"totalLength"`
	Files        []TorrentFile `json:"files"`
	Magnet       string        `json:"magnet"`
}

type TorrentFile struct {
	Length int64  `json:"length"`
	Path   string `json:"path"`
}

type InfoDict struct {
	Name        string `json:"name"`
	PieceLength int64  `json:"pieceLength"`
	Private     bool   `json:"private"`
	// Single-file mode:
	Length *int64 `json:"length,omitempty"`
	// Multi-file mode:
	Files []InfoFile `json:"files,omitempty"`
}

type InfoFile struct {
	Length int64    `json:"length"`
	Path   []string `json:"path"`
}

// -----------------------------
// Minimal bencode reader
// -----------------------------

type benReader struct {
	b   []byte
	pos int
}

func NewReader(b []byte) *benReader { return &benReader{b: b} }

func (r *benReader) Peek() (byte, error) {
	if r.pos >= len(r.b) {
		return 0, io.ErrUnexpectedEOF
	}
	return r.b[r.pos], nil
}

func (r *benReader) readByte() (byte, error) {
	if r.pos >= len(r.b) {
		return 0, io.ErrUnexpectedEOF
	}
	ch := r.b[r.pos]
	r.pos++
	return ch, nil
}

func (r *benReader) expectByte(want byte) error {
	ch, err := r.readByte()
	if err != nil {
		return err
	}
	if ch != want {
		return fmt.Errorf("bencode: expected '%c' got '%c' at %d", want, ch, r.pos-1)
	}
	return nil
}

func (r *benReader) readInt() (int64, error) {
	// i<digits>e
	if err := r.expectByte('i'); err != nil {
		return 0, err
	}
	sign := int64(1)
	if r.pos < len(r.b) && r.b[r.pos] == '-' {
		sign = -1
		r.pos++
	}
	if r.pos >= len(r.b) || r.b[r.pos] < '0' || r.b[r.pos] > '9' {
		return 0, fmt.Errorf("bencode: invalid integer at %d", r.pos)
	}
	var n int64
	for {
		if r.pos >= len(r.b) {
			return 0, io.ErrUnexpectedEOF
		}
		ch := r.b[r.pos]
		if ch == 'e' {
			r.pos++
			break
		}
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("bencode: invalid digit at %d", r.pos)
		}
		n = n*10 + int64(ch-'0')
		r.pos++
	}
	return sign * n, nil
}

func (r *benReader) readString() ([]byte, error) {
	// <len>:<data>
	if r.pos >= len(r.b) || r.b[r.pos] < '0' || r.b[r.pos] > '9' {
		return nil, fmt.Errorf("bencode: expected string length at %d", r.pos)
	}

	var n int
	for {
		if r.pos >= len(r.b) {
			return nil, io.ErrUnexpectedEOF
		}

		ch := r.b[r.pos]
		if ch == ':' {
			r.pos++
			break
		}

		if ch < '0' || ch > '9' {
			return nil, fmt.Errorf("bencode: invalid length at %d", r.pos)
		}

		n = n*10 + int(ch-'0')
		r.pos++
	}

	if r.pos+n > len(r.b) {
		return nil, io.ErrUnexpectedEOF
	}

	data := r.b[r.pos : r.pos+n]
	r.pos += n
	return data, nil
}

// skipAny advances r.pos past the next value
func (r *benReader) skipAny() error {
	ch, err := r.Peek()
	if err != nil {
		return err
	}

	switch ch {
	case 'i':
		_, err := r.readInt()
		return err

	case 'l':
		_, _ = r.readByte() // 'l'
		for {
			ch, err := r.Peek()
			if err != nil {
				return err
			}
			if ch == 'e' {
				_, _ = r.readByte()
				return nil
			}
			if err := r.skipAny(); err != nil {
				return err
			}
		}

	case 'd':
		_, _ = r.readByte()
		for {
			ch, err := r.Peek()
			if err != nil {
				return err
			}
			if ch == 'e' {
				_, _ = r.readByte()
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
		// string
		_, err := r.readString()
		return err
	}
}

// -----------------------------
// Decoding specific torrent fields while preserving raw span of info dict
// -----------------------------

type span struct{ start, end int }

type metaTop struct {
	announce     string
	announceList [][]string
	creationDate *time.Time
	comment      string
	createdBy    string
	encoding     string
	info         InfoDict
	infoSpan     span // raw bytes covering the bencoded info dictionary
	piecesRaw    []byte
}

func DecodeTorrent(b []byte) (*metaTop, error) {
	r := NewReader(b)
	// top-level must be dict
	if err := r.expectByte('d'); err != nil {
		return nil, err
	}

	var mt metaTop
	for {
		ch, err := r.Peek()
		if err != nil {
			return nil, err
		}

		if ch == 'e' {
			_, _ = r.readByte()
			break
		}

		keyb, err := r.readString()

		if err != nil {
			return nil, err
		}

		key := string(keyb)
		switch key {
		case "announce":
			v, err := r.readString()
			if err != nil {
				return nil, err
			}
			mt.announce = string(v)

		case "announce-list":
			al, err := ReadStringListOfLists(r)
			if err != nil {
				return nil, fmt.Errorf("announce-list: %w", err)
			}
			mt.announceList = al

		case "creation date":
			iv, err := r.readInt()
			if err != nil {
				return nil, err
			}
			t := time.Unix(iv, 0).UTC()
			mt.creationDate = &t

		case "comment":
			v, err := r.readString()
			if err != nil {
				return nil, err
			}
			mt.comment = string(v)

		case "created by":
			v, err := r.readString()
			if err != nil {
				return nil, err
			}
			mt.createdBy = string(v)

		case "encoding":
			v, err := r.readString()
			if err != nil {
				return nil, err
			}
			mt.encoding = string(v)

		case "info":
			// capture raw span exactly as encoded
			start := r.pos
			if err := r.skipAny(); err != nil {
				return nil, err
			}
			end := r.pos

			// Decode the info dict by re-parsing this span
			sub := NewReader(b[start:end])
			info, piecesRaw, err := DecodeInfo(sub)
			if err != nil {
				return nil, err
			}

			mt.info = info
			mt.infoSpan = span{start, end}
			mt.piecesRaw = piecesRaw

		default:
			// unknown key: skip value
			if err := r.skipAny(); err != nil {
				return nil, err
			}
		}
	}

	return &mt, nil
}

func ReadStringListOfLists(r *benReader) ([][]string, error) {
	if err := r.expectByte('l'); err != nil {
		return nil, err
	}
	var out [][]string
	for {
		ch, err := r.Peek()
		if err != nil {
			return nil, err
		}
		if ch == 'e' {
			_, _ = r.readByte()
			break
		}
		// inner list
		if err := r.expectByte('l'); err != nil {
			return nil, err
		}
		var inner []string
		for {
			ch, err := r.Peek()
			if err != nil {
				return nil, err
			}
			if ch == 'e' {
				_, _ = r.readByte()
				break
			}
			v, err := r.readString()
			if err != nil {
				return nil, err
			}
			inner = append(inner, string(v))
		}
		out = append(out, inner)
	}
	return out, nil
}

func DecodeInfo(r *benReader) (InfoDict, []byte, error) {
	var info InfoDict
	var piecesRaw []byte
	if err := r.expectByte('d'); err != nil {
		return info, nil, err
	}

	for {
		ch, err := r.Peek()
		if err != nil {
			return info, nil, err
		}

		if ch == 'e' {
			_, _ = r.readByte()
			break
		}

		keyb, err := r.readString()
		if err != nil {
			return info, nil, err
		}

		key := string(keyb)
		switch key {
		case "name":
			v, err := r.readString()
			if err != nil {
				return info, nil, err
			}
			info.Name = string(v)

		case "piece length":
			iv, err := r.readInt()
			if err != nil {
				return info, nil, err
			}
			info.PieceLength = iv

		case "private":
			iv, err := r.readInt()
			if err != nil {
				return info, nil, err
			}
			info.Private = (iv != 0)

		case "length":
			iv, err := r.readInt()
			if err != nil {
				return info, nil, err
			}
			info.Length = &iv

		case "files":
			files, err := DecodeInfoFiles(r)
			if err != nil {
				return info, nil, err
			}
			info.Files = files

		case "pieces":
			v, err := r.readString()
			if err != nil {
				return info, nil, err
			}
			piecesRaw = append([]byte(nil), v...) // copy

		default:
			if err := r.skipAny(); err != nil {
				return info, nil, err
			}
		}
	}
	return info, piecesRaw, nil
}

func DecodeInfoFiles(r *benReader) ([]InfoFile, error) {
	if err := r.expectByte('l'); err != nil {
		return nil, err
	}
	var out []InfoFile
	for {
		ch, err := r.Peek()
		if err != nil {
			return nil, err
		}
		if ch == 'e' {
			_, _ = r.readByte()
			break
		}
		if err := r.expectByte('d'); err != nil {
			return nil, err
		}
		var f InfoFile
		for {
			ch, err := r.Peek()
			if err != nil {
				return nil, err
			}
			if ch == 'e' {
				_, _ = r.readByte()
				break
			}
			keyb, err := r.readString()
			if err != nil {
				return nil, err
			}
			key := string(keyb)
			switch key {
			case "length":
				iv, err := r.readInt()
				if err != nil {
					return nil, err
				}
				f.Length = iv
			case "path":
				paths, err := ReadStringList(r)
				if err != nil {
					return nil, err
				}
				f.Path = paths
			default:
				if err := r.skipAny(); err != nil {
					return nil, err
				}
			}
		}
		out = append(out, f)
	}
	return out, nil
}

func ReadStringList(r *benReader) ([]string, error) {
	if err := r.expectByte('l'); err != nil {
		return nil, err
	}
	var out []string
	for {
		ch, err := r.Peek()
		if err != nil {
			return nil, err
		}
		if ch == 'e' {
			_, _ = r.readByte()
			break
		}
		v, err := r.readString()
		if err != nil {
			return nil, err
		}
		out = append(out, string(v))
	}
	return out, nil
}

// -----------------------------
// Assembly / helpers
// -----------------------------

func AssembleTorrent(b []byte) (*Torrent, error) {
	mt, err := DecodeTorrent(b)
	if err != nil {
		return nil, err
	}

	if mt.infoSpan.start <= 0 || mt.infoSpan.end <= mt.infoSpan.start || mt.infoSpan.end > len(b) {
		return nil, errors.New("invalid info dictionary span")
	}

	// info-hash (exact raw bytes of info dict)
	h := sha1.Sum(b[mt.infoSpan.start:mt.infoSpan.end])

	// pieces parsing
	if len(mt.piecesRaw) == 0 || len(mt.piecesRaw)%20 != 0 {
		return nil, fmt.Errorf("invalid pieces field length: %d", len(mt.piecesRaw))
	}

	pieceCount := len(mt.piecesRaw) / 20
	pieceHashes := make([]string, 0, pieceCount)
	for i := range pieceCount {
		pieceHashes = append(pieceHashes, hex.EncodeToString(mt.piecesRaw[i*20:(i+1)*20]))
	}

	// files + total length
	var files []TorrentFile
	var total int64
	if mt.info.Length != nil {
		// single-file
		files = []TorrentFile{{Length: *mt.info.Length, Path: mt.info.Name}}
		total = *mt.info.Length
	} else {
		for _, f := range mt.info.Files {
			p := filepath.ToSlash(filepath.Join(append([]string{mt.info.Name}, f.Path...)...))
			files = append(files, TorrentFile{Length: f.Length, Path: p})
			total += f.Length
		}
		// deterministic order for output
		sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	}
	announceList := mt.announceList
	if mt.announce != "" {
		// ensure primary announce is first entry if not already present
		found := false
		for _, tier := range announceList {
			for _, tr := range tier {
				if tr == mt.announce {
					found = true
					break
				}
			}
		}
		if !found {
			announceList = append([][]string{{mt.announce}}, announceList...)
		}
	}
	infoHashHex := hex.EncodeToString(h[:])
	infoHashB32 := strings.TrimRight(base32.StdEncoding.EncodeToString(h[:]), "=")
	magnet := BuildMagnet(h[:], announceList)

	return &Torrent{
		Announce:     mt.announce,
		AnnounceList: announceList,
		CreationDate: mt.creationDate,
		Comment:      mt.comment,
		CreatedBy:    mt.createdBy,
		Encoding:     mt.encoding,
		Info:         mt.info,
		InfoHash:     infoHashHex,
		InfoHashB32:  infoHashB32,
		PieceHashes:  pieceHashes,
		TotalLength:  total,
		Files:        files,
		Magnet:       magnet,
	}, nil
}

func BuildMagnet(infoHash []byte, announceList [][]string) string {
	xt := "urn:btih:" + strings.TrimRight(base32.StdEncoding.EncodeToString(infoHash), "=")
	var parts []string
	parts = append(parts, "xt="+xt)
	// include a few top trackers (dedup)
	seen := map[string]struct{}{}
	count := 0
	for _, tier := range announceList {
		for _, tr := range tier {
			if _, ok := seen[tr]; ok {
				continue
			}
			seen[tr] = struct{}{}
			parts = append(parts, "tr="+EscapeMagnet(tr))
			count++
			if count >= 8 {
				break
			}
		}
		if count >= 8 {
			break
		}
	}
	return "magnet:?" + strings.Join(parts, "&")
}

func EscapeMagnet(s string) string {
	// very small percent-encoder for trackers in magnets
	replacer := strings.NewReplacer(
		"%", "%25",
		" ", "%20",
		"\"", "%22",
		"<", "%3C",
		">", "%3E",
		"#", "%23",
		"|", "%7C",
	)
	return replacer.Replace(s)
}

// -----------------------------
// CLI
// -----------------------------

func CLI() {
	var outJSON bool
	flag.BoolVar(&outJSON, "json", true, "print JSON summary (default true)")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: torrentparse [--json] file.torrent")
		os.Exit(2)
	}

	path := flag.Arg(0)
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}

	t, err := AssembleTorrent(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse:", err)
		os.Exit(1)
	}

	if outJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(t); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	// pretty print
	fmt.Printf("Announce: %s\n", t.Announce)
	if len(t.AnnounceList) > 0 {
		fmt.Println("Trackers:")
		for i, tier := range t.AnnounceList {
			fmt.Printf("  Tier %d:\n", i+1)
			for _, tr := range tier {
				fmt.Printf("    %s\n", tr)
			}
		}
	}
	if t.CreationDate != nil {
		fmt.Printf("Creation date: %s\n", t.CreationDate.Format(time.RFC3339))
	}
	if t.Comment != "" {
		fmt.Printf("Comment: %s\n", t.Comment)
	}
	if t.CreatedBy != "" {
		fmt.Printf("Created by: %s\n", t.CreatedBy)
	}
	fmt.Printf("Info hash (hex): %s\n", t.InfoHash)
	fmt.Printf("Info hash (base32): %s\n", t.InfoHashB32)
	fmt.Printf("Name: %s\n", t.Info.Name)
	fmt.Printf("Piece length: %d\n", t.Info.PieceLength)
	fmt.Printf("Private: %v\n", t.Info.Private)
	fmt.Printf("Total length: %d bytes\n", t.TotalLength)
	fmt.Printf("Files: %d\n", len(t.Files))
	for _, f := range t.Files {
		fmt.Printf("  %12d  %s\n", f.Length, f.Path)
	}
	fmt.Printf("Magnet: %s\n", t.Magnet)
}
