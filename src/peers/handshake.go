package peers

import (
	"fmt"
	"net"
)

type initTCPconn struct {
	host net.IP
	port uint32
}

func StartConnections(peers []Peer) error {
	for _, peer := range peers {
		lis, err := net.Listen("tcp", peer.IP.String()+":"+string(peer.Port))
		if err != nil {
			return err
		}

		fmt.Println(lis)
	}
	return nil
}
