package peers

import (
	"fmt"
	"net"
	"torrent-client/parser"
)

type initTCPconn struct {
	host net.IP
	port uint32
}

func StartPeerConnections(peers []parser.Peer) error {
	for _, peer := range peers {
		lis, err := net.Listen("tcp", peer.Ip.String()+":"+string(peer.Port))
		if err != nil {
			return err
		}

		fmt.Println(lis)
	}
	return nil
}
