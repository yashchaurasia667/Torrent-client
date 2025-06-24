package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	announce := "udp://tracker.openbittorrent.com:80/announce"
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Announce URL [default: %s]: ", announce)
	fmt.Scanln(&announce)

	fmt.Printf("Author: ")
	author, _ := reader.ReadString('\n')
	author = strings.TrimSpace(author)

	creationDate := time.Now().Unix()
	encoding := "UTF-8"

	fmt.Println("Comment [default: \"\"]: ")
	comment, _ := reader.ReadString('\n')
	comment = strings.TrimSpace(comment)

	var name string
	for {
		fmt.Println("Name of the file [essential]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input: ", err)
			continue
		}
		name = strings.TrimSpace(input)

		if name != "" {
			break
		}
	}

	pieceSize := 2
	fmt.Printf("Piece Size [default: %dMB]: ", &pieceSize)
	_, err := fmt.Scan(&pieceSize)

	if err != nil {
		fmt.Println("Invalid input:", err)
	}

}
