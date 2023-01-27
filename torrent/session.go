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

type status struct {
	pieces []pieceStatus
	mutex  sync.Mutex
}

type pieceStatus int

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
	_, err = rw.Write(handshakeMsg())
	if err != nil {
		return fmt.Errorf("handshake failed: %v", err)
	}
	err = readHandshake(rw.Reader)
	if err != nil {
		return fmt.Errorf("failed to read responce handshake: %v", err)
	}
	_, err = rw.Write(interestedMsg())
	if err != nil {
		return fmt.Errorf("failed to send 'interested' message: %v", err)
	}
	for {
		msg, err := readMessage(rw.Reader)
		if err != nil {
			return err
		}
		switch msg.id {
		case choke:
			s.handleChoke()
		case unchoke:
			s.handleUnchoke()
		case have:
			s.handleHave()
		case piece:
			s.handlePiece()
		}
	}
}

const (
	needed pieceStatus = iota
	requested
	received
)

func handshakeMsg() []byte {
	// TODO
	return nil
}

func readHandshake(r *bufio.Reader) error {
	pstrlenBuf := make([]byte, 1)
	_, err := io.ReadFull(r, pstrlenBuf)
	if err != nil {
		return err
	}
	pstrlen := pstrlenBuf[0]
	restBuf := make([]byte, pstrlen+8+20)
	_, err = io.ReadFull(r, restBuf)
	return err
}

func interestedMsg() []byte {
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

func (s *session) handleChoke() {

}

func (s *session) handleUnchoke() {

}

func (s *session) handleHave() {

}

func (s *session) handlePiece() {

}
