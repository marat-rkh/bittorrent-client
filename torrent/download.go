package torrent

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
)

type Download struct {
	torrent     *File
	trackerResp *TrackerResponse
	status      *status
}

func NewDownload(torrent *File, resp *TrackerResponse) *Download {
	return &Download{torrent: torrent, trackerResp: resp}
}

type status struct {
	pieces []pieceStatus
	mutex  sync.Mutex
}

type pieceStatus int

const (
	needed pieceStatus = iota
	requested
	received
)

func (d *Download) Start() {
	for i := range d.trackerResp.Peers {
		// TODO should we try buffered chan here? should we use total num of pieces as a buffer size?
		s := session{download: d, peerIdx: i, piecesQueue: make(chan int)}
		// TODO handle error
		go s.start()
	}
}

type session struct {
	download    *Download
	peerIdx     int
	piecesQueue chan int
}

func (s *session) start() error {
	peer := s.download.trackerResp.Peers[s.peerIdx]
	addr := peer.IP + ":" + strconv.Itoa(int(peer.Port))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to peer: %v", err)
	}
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	_, err = rw.Write(handshake())
	if err != nil {
		return fmt.Errorf("handshake failed: %v", err)
	}
	go s.requestPieces()
	for {
		msg, err := readMessage(rw.Reader)
		if err != nil {
			return err
		}
		switch msg.id {
		}
	}
}

func handshake() []byte {
	// TODO
	return nil
}

func readMessage(r *bufio.Reader) (message, error) {
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return message{}, err
	}
	length := binary.BigEndian.Uint32(lengthBuf)
	if length == 0 {
		return message{id: keepAlive}, nil
	}
	msgBuf := make([]byte, length)
	_, err = io.ReadFull(r, msgBuf)
	if err != nil {
		return message{}, err
	}
	return message{length: length, id: messageId(msgBuf[0]), payload: msgBuf[1:]}, err
}

type message struct {
	length  uint32
	id      messageId
	payload []byte
}

type messageId int

const (
	keepAlive messageId = iota - 1
	choke
	unchoke
	interested
	notInterested
	have
	bitfield
	request
	piece
	cancel
	port
)

func (s *session) requestPieces() {
	// TODO we should close `s.piecesQueue` when all values in `Download.status.values` are `received`
	// for piece := range s.piecesQueue {}
}
