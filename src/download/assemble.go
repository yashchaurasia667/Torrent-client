package download

import (
	"fmt"
	"os"
	"path/filepath"
)

func WritePiece(pieceIndex uint32, piece []byte, outDir string) error {
	fileName := fmt.Sprintf("piece%d.part", pieceIndex)
	fullPath := filepath.Join(outDir, fileName)

	fmt.Println(fullPath)

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
