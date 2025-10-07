package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"torrent-client/src/download"
	"torrent-client/src/parser"
	"torrent-client/src/peers"
	"torrent-client/src/utils"
)

const CONCURRENT_DONWLOADS = 10

func check(path string, outDir string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Invalid file path.")
		os.Exit(1)
	}
	parts := strings.Split(path, ".")
	if parts[len(parts)-1] != "torrent" {
		fmt.Fprintln(os.Stderr, "The file passed is not a torrent file.")
		os.Exit(1)
	}

	err := os.MkdirAll(outDir, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create out directory.")
		os.Exit(1)
	}
}

func getDownloadedLen(pieceCount uint32) uint32 {
	len := pieceCount / 8
	if pieceCount%8 != 0 {
		len += 1
	}
	return len
}

func GetPeers(t *parser.Torrent) (*parser.Response, error) {
	res, err := peers.RequestTracker(t, t.Announce)
	if err == nil {
		return res, nil
	} else {
	}

	for _, url := range t.AnnounceList {
		err = nil
		res, err = peers.RequestTracker(t, url)
		if err == nil {
			return res, nil
		} else {
		}
	}

	return nil, err
}

func getNextPieceIndex(downloaded []byte, bitField []byte, downloading *utils.DownloadingSet) (int, int, uint32, error) {
	for {
		dIndex, bIndex, err := download.GetNextDownloadablePiece(bitField, downloaded)
		if err != nil {
			return 0, 0, 0, err
		}
		pieceIndex := uint32(dIndex*8 + bIndex)
		if downloading.Contains(pieceIndex) {
			downloaded[dIndex] += byte(1 << (7 - bIndex))
			continue
		}
		return dIndex, bIndex, pieceIndex, nil
	}
}

/*
This function will be called asynchrousnously

Args=>
peer: struct containing info about peer that the tracker sent (IP, Port)
t: struct containing parsed torrent information
downloaded: slice containing info about all the pieces that have been downloaded
downloading: channel containing info about all the pieces that are currently downloading
wg: waitgroup to create a joining point to the main function
*/

func HandshakeNDownload(peer parser.Peer, t *parser.Torrent, downloaded *utils.Downloaded, peerId []byte, downloading *utils.DownloadingSet, outDir string, peerList []parser.Peer) error {
	// var pieces []utils.DownloadResult
	c, err := peers.PerformHandshake(peer, t.InfoHash, peerId)
	if err != nil || c == nil {
		return err
	}
	defer c.Conn.Close()

	intr := peers.CheckInterested(c.Conn)
	if !intr {
		c.Conn.Close()
		return fmt.Errorf("%s is not interested", peer.Ip.String())
	}
	fmt.Printf("%s has unchoked you. Now requesting a piece\n", peer.Ip.String())

	// download all the available pieces that peer offers
	for {
		tmp := append([]byte(nil), downloaded.GetContent()...)
		dIndex, bIndex, pieceIndex, err := getNextPieceIndex(tmp, c.Bitfield, downloading)
		if err != nil {
			return err
		}
		// downloaded[dIndex] += byte(1 << (7 - bIndex))
		downloaded.Add(dIndex, bIndex)
		downloading.Add(pieceIndex)

		piece, err := download.DownloadPiece(c.Conn, c.Bitfield, t, pieceIndex)
		if err != nil {
			// downloaded[dIndex] -= byte(1 << (7 - bIndex))
			downloaded.Remove(dIndex, bIndex)
			break
		}
		fmt.Printf("Downloaded piece index %d from peer %s\n", pieceIndex, peer.Ip.String())
		// pieces = append(pieces, utils.DownloadResult{DIndex: dIndex, BIndex: bIndex})
		downloading.Remove(pieceIndex)
		peers.SendHavePiece(peerList, pieceIndex)

		err = download.WritePiece(pieceIndex, piece, filepath.Join(outDir, t.Info.Name))
		if err != nil {
			// downloaded[dIndex] -= byte(1 << (7 - bIndex))
			downloaded.Remove(dIndex, bIndex)
			return err
		}
	}
	return nil
}

func main() {
	/*
		args => command line arguments
		wg => wait group to synchronize main with handshakeNdownload
		downloading => map to mark the pieces that are downloading
		threadLimit => to limit maximum concurrent downloads to value of CONCURRENT_DOWNLOADS
	*/
	args := os.Args
	var wg sync.WaitGroup
	downloading := utils.NewDownloadingSet()
	threadLimit := make(chan struct{}, CONCURRENT_DONWLOADS)

	// Exit if no file path is passed
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: ./torrent-client [file path] [out path]")
		os.Exit(1)
	}
	// check for file and path validity
	check(args[1], args[2])

	// Read file and Get decoded struct
	data, err := os.ReadFile(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Read: ", err)
		os.Exit(1)
	}

	t, err := parser.AssembleTorrent(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error while reading torrent file: ", err)
		os.Exit(1)
	}

	downloaded := utils.NewDownloaded(getDownloadedLen(t.Info.PieceCount))
	peerId := peers.GetPeerId()
	for {
		// 1. get peers from traker
		fmt.Println("Requesting a fresh list of peers from the tracker")
		res, err := GetPeers(t)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get peers from any tracker: %v\n", err)
			time.Sleep(15 * time.Second)
			continue
		}
		fmt.Printf("Total Length: %d, Piece Length: %d, block size: %d, Piece Count: %d\n", t.TotalLength, t.Info.PieceLength, download.BLOCK_SIZE, t.Info.PieceCount)

		// 2. send interested to all the peers and wait for unchoke
		// 3. when unchoked get the bitfield and get next downloadable piece
		if len(res.Peers) == 0 {
			fmt.Println("No peers found, retrying in 30 seconds...")
			time.Sleep(30 * time.Second)
			continue
		}

		for _, peer := range res.Peers {
			if peer.Ip.String() == "0.0.0.0" {
				continue
			}

			wg.Add(1)
			threadLimit <- struct{}{}
			go func(p parser.Peer) {
				defer func() {
					<-threadLimit
					wg.Done()
				}()

				// 4. download the piece
				err := HandshakeNDownload(p, t, downloaded, []byte(peerId), downloading, args[2], res.Peers)
				if err != nil {
					if err == io.EOF {
						fmt.Fprintln(os.Stderr, "Error:", peer.Ip.String(), "dropped connection")
						return
					}
					fmt.Printf("Error: %s\n", err)
					return
				}
			}(peer)
		}

		// wait for all downloads in this batch to complete
		wg.Wait()

		// 5. after you've gone through all the peers but you still dont have all the pieces repeat all the steps again
		if downloaded.GetPieceCount() == t.Info.PieceCount {
			break
		}

		// wg.Wait()
	}
}
