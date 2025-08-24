package parser

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
	Interval int
	Peers    []Peer
}

type Peer struct {
	Ip   string
	Port int
}

func (r *Reader) decodePeer() (*Peer, error) {
	var peer Peer
	for {
		err := r.expectByte('d')
		if err != nil {
			return nil, err
		}

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
			peer.Ip = ip
		case "port":
			port, err := r.readInt()
			if err != nil {
				return nil, err
			}
			peer.Port = int(port)
		}
	}
	return &peer, nil
}

func (r *Reader) decodePeers() ([]Peer, error) {
	var p []Peer

	for {
		err := r.expectByte('l')
		if err != nil {
			return nil, err
		}

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

func (r *Reader) DecodeResponse(b []byte) (*Response, error) {
	var res Response

	for {
		err := r.expectByte('d')
		if err != nil {
			return nil, err
		}

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
			res.Interval = int(i)

		case "peers":
			// var p []Peer
			p, err := r.decodePeers()
			if err != nil {
				return nil, err
			}
			res.Peers = p
		}
	}

	return &res, nil
}
