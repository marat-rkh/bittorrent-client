package torrent

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"github.com/zeebo/bencode"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// File is a metainfo file.
// See: https://wiki.theory.org/index.php/BitTorrentSpecification#Metainfo_File_Structure
type File struct {
	Announce string `bencode:"announce"`
	Info     struct {
		PieceLen int64  `bencode:"piece length"`
		Pieces   string `bencode:"pieces"`
		Length   int64  `bencode:"length"`
		Files    []struct {
			Length int64    `bencode:"length"`
			Path   []string `bencode:"path"`
		}
	} `bencode:"info"`
}

// TrackerResponse contains data returned by the tracker upon the announce request.
// See: https://wiki.theory.org/index.php/BitTorrentSpecification#Tracker_Response
type TrackerResponse struct {
	FailureReason  string `bencode:"failure reason"`
	WarningMessage string `bencode:"warning message"`
	Interval       int    `bencode:"interval"`
	MinInterval    int    `bencode:"min interval"`
	TrackerId      string `bencode:"tracker id"`
	Complete       int    `bencode:"complete"`
	Incomplete     int    `bencode:"incomplete"`
	Peers          []struct {
		PeerId string `bencode:"peer id"`
		IP     string `bencode:"ip"`
		Port   int    `bencode:"port"`
	} `bencode:"peers"`
}

// Parse extract a metainfo from the torrent file specified by path argument.
func Parse(path string) (*File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open torrent file: %v\n", err)
	}
	dec := bencode.NewDecoder(file)
	var tFile File
	if err := dec.Decode(&tFile); err != nil {
		return nil, fmt.Errorf("failed to decode torrent file: %v\n", err)
	}
	return &tFile, nil
}

// MakeAnnounceRequest sends the announce request to the tracker to get info required to participate in torrent.
// In particular, it contains a list of peers with a file we want to download.
func (f *File) MakeAnnounceRequest() (*TrackerResponse, error) {
	u, err := url.Parse(f.Announce)
	if err != nil {
		return nil, fmt.Errorf("failed to parse announce URL: %v", err)
	}
	params, err := peersRequestParams(f)
	if err != nil {
		return nil, err
	}
	u.RawQuery = params.Encode()
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("get request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read responce body: %v", err)
	}
	body := bytes.NewBuffer(bodyBytes)
	log.Printf("responce body:\"%s\"", body.String())
	dec := bencode.NewDecoder(body)
	var tResp TrackerResponse
	err = dec.Decode(&tResp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tracker response: %v", err)
	}
	return &tResp, nil
}

// See: https://wiki.theory.org/index.php/BitTorrentSpecification#Tracker_Request_Parameters
func peersRequestParams(tFile *File) (url.Values, error) {
	params := url.Values{}
	infoHash, err := tFile.infoHash()
	if err != nil {
		return nil, fmt.Errorf("failed to calulate info hash: %v", err)
	}
	params.Add("info_hash", base64.StdEncoding.EncodeToString(infoHash))
	peerId := make([]byte, 20)
	peerId[0] = '-'
	peerId[1] = 'M'
	peerId[2] = 'K'
	peerId[3] = '0'
	peerId[4] = '1'
	peerId[5] = '0'
	peerId[6] = '0'
	peerId[7] = '-'
	_, err = rand.Read(peerId[8:])
	log.Printf("peer id: %s%v\n", string(peerId[0:8]), peerId[8:])
	if err != nil {
		return nil, fmt.Errorf("failed to generate peer id: %v", err)
	}
	params.Add("peer_id", base64.StdEncoding.EncodeToString(peerId))
	params.Add("left", strconv.FormatInt(tFile.length(), 10))
	// TODO check if port is available, automatically select the other if not available
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("compact", "0")
	params.Add("no_peer_id", "true")
	return params, nil
}

func (f *File) infoHash() ([]byte, error) {
	info := strings.Builder{}
	enc := bencode.NewEncoder(&info)
	err := enc.Encode(&f.Info)
	if err != nil {
		return nil, fmt.Errorf("failed to encode torrent info: %v", err)
	}
	hash := sha1.New()
	_, _ = io.WriteString(hash, info.String())
	return hash.Sum(nil), nil
}

func (f *File) length() int64 {
	if len(f.Info.Files) == 0 {
		return f.Info.Length
	}
	var res int64
	for _, file := range f.Info.Files {
		res += file.Length
	}
	return res
}
