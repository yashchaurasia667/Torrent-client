package download

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"torrent-client/src/parser"
)

type FileReader struct {
	FileName string
	Pos      uint32
}

func WritePiece(pieceIndex uint32, piece []byte, outDir string) error {
	fileName := fmt.Sprintf("piece%d.part", pieceIndex)
	fullPath := filepath.Join(outDir, fileName)

	// fmt.Println(fullPath)

	dir := filepath.Dir(fullPath)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not create necessary directory %s: %w", dir, err)
	}

	err = os.WriteFile(fullPath, piece, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}
	return nil
}

func singleFileWrite(outDir string, fileName string) error {
	fullPath := filepath.Join(outDir, fileName)
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		return fmt.Errorf("path %s does not exist", outDir)
	}

	pieces, err := os.ReadDir(outDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %s", outDir)
	}

	for _, piece := range pieces {
		piecePath := filepath.Join(outDir, piece.Name())
		data, err := os.ReadFile(piecePath)
		if err != nil {
			return fmt.Errorf("failed to read a file[%s]: %s", piecePath, err)
		}

		// Create the file first time when it does not exist
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			err := os.WriteFile(fullPath, data, os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to create file [%s]", fullPath)
			}
			continue
		}

		// append to the created file
		file, err := os.OpenFile(fullPath, os.O_APPEND, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to open file [%s]", fullPath)
		}
		defer file.Close()

		if _, err := file.Write(data); err != nil {
			return fmt.Errorf("failed to append to file [%s]", fullPath)
		}
	}
	return nil
}

/*
func AssembleTorrent(t *parser.Torrent, outDir string) error {
	files, err := os.ReadDir(outDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %s", outDir)
	}
	if len(files) != int(t.Info.PieceCount) {
		return fmt.Errorf("file not completely downloaded: expected piece count %d got %d", t.Info.PieceCount, len(files))
	}

	var f FileReader
	var pieceCounter uint32 = 0
	for _, infoFile := range t.Info.Files {
		var path = outDir
		f.FileName = infoFile.Path[len(infoFile.Path)-1]

		if len(infoFile.Path) > 1 {
			fullPath := []string{outDir}
			fullPath = append(fullPath, infoFile.Path...)
			path = filepath.Join(fullPath...)

			if _, err := os.Stat(path); os.IsNotExist(err) {
				err := os.MkdirAll(path, os.ModePerm)
				if err != nil {
					return err
				}
			}
		}

		for {
			var piecePath = filepath.Join(outDir, t.Info.Name, fmt.Sprintf("piece%d.part", pieceCounter))
			data, err := os.ReadFile(piecePath)
			if err != nil {
				return fmt.Errorf("failed to read a file[%s]: %s", piecePath, err)
			}

			if infoFile.Length < uint64(len(data)) {
				err = os.WriteFile(path, data, os.ModeAppend)
			} else {
				err = os.WriteFile(path, data, os.ModeAppend)
				f.Pos += uint32(len(data))
				pieceCounter++
			}
			if err != nil {
				return fmt.Errorf("failed to write a file[%s]: %s", f.FileName, err)
			}
		}
	}
	return nil
}
*/

func AssembleFiles(t *parser.Torrent, outDir string, deletePieces bool) error {
	if t == nil {
		return fmt.Errorf("torrent is nil")
	}

	baseDir := filepath.Join(outDir, t.Info.Name)
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return fmt.Errorf("output directory %s does not exist", baseDir)
	}

	pieceCount := int(t.Info.PieceCount)
	// pieceLen := int(t.Info.PieceLength)

	piecePath := func(index int) string {
		return filepath.Join(baseDir, fmt.Sprintf("piece%d.part", index))
	}

	readPiece := func(index int) ([]byte, error) {
		data, err := os.ReadFile(piecePath(index))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", piecePath(index), err)
		}
		return data, nil
	}

	var (
		pieceIndex   int
		pieceOffset  int
		currentPiece []byte
	)

	getBytes := func(n int) ([]byte, error) {
		var out []byte
		for n > 0 {
			if currentPiece == nil || pieceOffset >= len(currentPiece) {
				if pieceIndex >= pieceCount {
					return out, io.EOF
				}
				p, err := readPiece(pieceIndex)
				if err != nil {
					return nil, err
				}
				currentPiece = p
				pieceOffset = 0
				pieceIndex++
			}

			remain := len(currentPiece) - pieceOffset
			toRead := min(remain, n)

			out = append(out, currentPiece[pieceOffset:pieceOffset+toRead]...)
			pieceOffset += toRead
			n -= toRead
		}
		return out, nil
	}

	if !t.HasMultipleFiles {
		// Single-file torrent
		outputFile := filepath.Join(baseDir)
		out, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer out.Close()

		for pieceIndex < pieceCount {
			piece, err := readPiece(pieceIndex)
			if err != nil {
				return err
			}
			if _, err := out.Write(piece); err != nil {
				return fmt.Errorf("failed writing piece %d: %w", pieceIndex, err)
			}
			pieceIndex++
		}
	} else {
		// Multi-file torrent
		for _, f := range t.Info.Files {
			filePath := filepath.Join(baseDir, filepath.Join(f.Path...))
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", filePath, err)
			}

			out, err := os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", filePath, err)
			}

			toWrite := int(f.Length)
			for toWrite > 0 {
				chunk, err := getBytes(toWrite)
				if err != nil && err != io.EOF {
					out.Close()
					return fmt.Errorf("error reading bytes for %s: %w", filePath, err)
				}
				if len(chunk) == 0 {
					break
				}
				_, err = out.Write(chunk)
				if err != nil {
					out.Close()
					return fmt.Errorf("failed writing to %s: %w", filePath, err)
				}
				toWrite -= len(chunk)
			}
			out.Close()
		}
	}

	if deletePieces {
		for i := 0; i < pieceCount; i++ {
			path := piecePath(i)
			os.Remove(path)
		}
	}

	return nil
}
