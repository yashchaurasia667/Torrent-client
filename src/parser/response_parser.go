package parser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// d
// 8:intervali900e
// 5:peers
// l
// d
// 2:ip 14:31.216.102.126
// 4:port i16881e
// e
// e
// e

type Response struct {
	Interval uint32
	Peers    []Peer
}

type Peer struct {
	Ip       net.IP
	Port     uint16
	Conn     net.Conn
	PeerId   [20]byte
	Bitfield []byte
}

func (r *Reader) decodePeer() (*Peer, error) {
	var peer Peer
	err := r.expectByte('d')
	if err != nil {
		return nil, err
	}

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
		case "ip":
			ip, err := r.readString()
			if err != nil {
				return nil, err
			}
			peer.Ip = net.ParseIP(ip)
		case "port":
			port, err := r.readInt()
			if err != nil {
				return nil, err
			}
			peer.Port = uint16(port)
		}
	}
	return &peer, nil
}

func (r *Reader) decodePeers() ([]Peer, error) {
	var p []Peer

	err := r.expectByte('l')
	if err != nil {
		return nil, err
	}

	for {
		ch, err := r.peek()
		if err != nil {
			return nil, err
		}

		if ch == 'e' {
			r.readByte()
			break
		}

		peer, err := r.decodePeer()
		if err != nil {
			return nil, err
		}

		p = append(p, *peer)
	}

	return p, nil
}

func (r *Reader) DecodeHttpResponse() (*Response, error) {
	var res Response

	err := r.expectByte('d')
	if err != nil {
		return nil, err
	}

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
		case "interval":
			i, err := r.readInt()
			if err != nil {
				return nil, err
			}
			res.Interval = uint32(i)

		case "peers":
			p, err := r.decodePeers()
			if err != nil {
				return nil, err
			}
			res.Peers = p
		}
	}

	return &res, nil
}

func DecodeUDPResponse(peersBin []byte) ([]Peer, error) {
	if len(peersBin)%6 != 0 {
		return nil, fmt.Errorf("invalid peers list length: %d", len(peersBin))
	}
	numPeers := len(peersBin) / 6
	peers := make([]Peer, 0, numPeers)
	reader := bytes.NewReader(peersBin)

	for range numPeers {
		var peer struct {
			IP   [4]byte
			Port uint16
		}
		if err := binary.Read(reader, binary.BigEndian, &peer); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		p := Peer{
			Ip:   net.IP(peer.IP[:]),
			Port: peer.Port,
		}
		peers = append(peers, p)
	}
	return peers, nil
}
