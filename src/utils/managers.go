package utils

import (
	// "fmt"
	"sync"
	"torrent-client/src/parser"
)

type DownloadingSet struct {
	mu sync.RWMutex
	m  map[uint32]struct{}
}

type Downloaded struct {
	mu         sync.RWMutex
	content    []byte
	pieceCount uint32
}

type DownloadResult struct {
	DIndex int
	BIndex int
}

type AvailablePeers struct {
	mu    sync.RWMutex
	peers map[string]parser.Peer
}

/* ---------- DOWNOLADING SET FUNCTIONS ---------- */

func NewDownloadingSet() *DownloadingSet {
	return &DownloadingSet{m: make(map[uint32]struct{})}
}

func (s *DownloadingSet) Add(idx uint32) {
	// fmt.Println("Downloading", idx)
	s.mu.Lock()
	s.m[idx] = struct{}{}
	s.mu.Unlock()
}

func (s *DownloadingSet) Remove(idx uint32) {
	// fmt.Printf("Removed %d from downloading\n", idx)
	s.mu.Lock()
	delete(s.m, idx)
	s.mu.Unlock()
}

func (s *DownloadingSet) Contains(idx uint32) bool {
	s.mu.Lock()
	_, ok := s.m[idx]
	s.mu.Unlock()
	return ok
}

/* ----------- DOWNLOADED FUNCTIONS ------------ */
func NewDownloaded(length uint32) *Downloaded {
	return &Downloaded{content: make([]byte, length)}
}

func (s *Downloaded) SetAll(pieceIndex uint32) {
	s.mu.Lock()
	for i := range s.content {
		if (i+1)*8 >= int(pieceIndex) {
			continue
		}
		s.content[i] = 255
	}
	s.mu.Unlock()
}

func (s *Downloaded) Add(dIndex int, bIndex int) {
	s.mu.Lock()
	bit := byte(1 << (7 - bIndex))
	// s.pieceCount += 1
	if s.content[dIndex]&bit == 0 {
		s.content[dIndex] |= bit
		s.pieceCount++
	}
	s.mu.Unlock()
}

func (s *Downloaded) Remove(dIndex int, bIndex int) {
	s.mu.Lock()
	bit := byte(1 << (7 - bIndex))
	if s.content[dIndex]&bit != 0 {
		s.content[dIndex] &^= bit
		s.pieceCount--
	}
	s.mu.Unlock()
}

func (s *Downloaded) GetContent() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	copySlice := make([]byte, len(s.content))
	copy(copySlice, s.content)
	return copySlice
}

func (s *Downloaded) GetPieceCount() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pieceCount
}

/*--------------------- PEER FUNCTIONS -----------------------*/
func NewPeerList(peerList []parser.Peer) *AvailablePeers {
	var av AvailablePeers
	for _, peer := range peerList {
		av.peers[peer.Ip.String()] = peer
	}
	return &av
}

func (s *AvailablePeers) Add(peer parser.Peer) {
	s.mu.Lock()
	s.peers[peer.Ip.String()] = peer
	s.mu.Unlock()
}

func (s *AvailablePeers) Remove(peer parser.Peer) {
	s.mu.Lock()
	delete(s.peers, peer.Ip.String())
	s.mu.Unlock()
}
