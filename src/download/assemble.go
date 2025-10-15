package download

import (
	"fmt"
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

func AssembleTorrent(t *parser.Torrent, outDir string) error {
	files, err := os.ReadDir(outDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %s\n", outDir)
	}
	if len(files) != int(t.Info.PieceCount) {
		return fmt.Errorf("file not completely downloaded: expected piece count %d got %d", t.Info.PieceCount, len(files))
	}

	var f FileReader
	for _, infoFile := range t.Info.Files {
		var path = outDir
		f.FileName = infoFile.Path[len(infoFile.Path)-1]

		if len(infoFile.Path) > 1 {
			parentDirs := []string{outDir}
			parentDirs = append(parentDirs, infoFile.Path[:len(infoFile.Path)-1]...)
			path = filepath.Join(parentDirs...)

			if _, err := os.Stat(path); os.IsNotExist(err) {
				err := os.MkdirAll(path, 0644)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
