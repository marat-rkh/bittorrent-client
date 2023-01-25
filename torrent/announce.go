package torrent

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/zeebo/bencode"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// MakeAnnounceRequest sends the announce request to the tracker to get info required to participate in torrent.
// In particular, it contains a list of peers with a file we want to download.
func (f *File) MakeAnnounceRequest() (*TrackerResponse, error) {
	u, err := url.Parse(f.announce)
	if err != nil {
		return nil, fmt.Errorf("failed to parse announce URL: %v", err)
	}
	params, err := peersRequestParams(f)
	if err != nil {
		return nil, err
	}
	u.RawQuery = params.Encode()
	fullUrl := u.String()
	log.Printf("url: %s", fullUrl)
	resp, err := http.Get(fullUrl)
	if err != nil {
		return nil, fmt.Errorf("get request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read responce body: %v", err)
	}
	body := bytes.NewBuffer(bodyBytes)
	log.Printf("responce body: \"%s\"\n", body.String())
	return parseResponse(body)
}

func parseResponse(resp *bytes.Buffer) (*TrackerResponse, error) {
	dec := bencode.NewDecoder(resp)
	var respMap map[string]interface{}
	err := dec.Decode(&respMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse announce response: %v", err)
	}
	log.Printf("decoded responce: %+v\n", respMap)
	if fail, ok := respMap["failure reason"]; ok {
		return nil, fmt.Errorf("announce request failed: %v", fail)
	}
	var tResp TrackerResponse
	if warn, ok := respMap["warning message"]; ok {
		tResp.WarningMessage = warn.(string)
	}
	interval, ok := respMap["interval"]
	if !ok {
		return nil, fmt.Errorf("cannot find 'interval' field in the announce responce")
	}
	tResp.Interval = interval.(int64)
	if minInterval, ok := respMap["min interval"]; ok {
		tResp.MinInterval = minInterval.(int64)
	}
	if tID, ok := respMap["tracker id"]; ok {
		tResp.TrackerId = tID.(string)
	}

	if complete, ok := respMap["complete"]; ok {
		tResp.Complete = complete.(int64)
	}
	if incomplete, ok := respMap["incomplete"]; ok {
		tResp.Incomplete = incomplete.(int64)
	}
	tResp.Peers, err = parsePeers(respMap)
	if err != nil {
		return nil, err
	}
	return &tResp, nil
}

func parsePeers(respMap map[string]interface{}) ([]PeerInfo, error) {
	peers, ok := respMap["peers"]
	if !ok {
		return nil, fmt.Errorf("cannot find 'peers' field in the announce responce")
	}
	peersStr, ok := peers.(string)
	if !ok {
		return nil, fmt.Errorf("'peers' field in the announce responce is not a string (dictinary mode is not supported yet)")
	}
	peersBytes := []byte(peersStr)
	if len(peersBytes)%6 != 0 {
		return nil, fmt.Errorf("'peers' field in the announce responce has incorrect size, must be N * 6")
	}
	var peersList []PeerInfo
	for i := 0; i < len(peersBytes); i += 6 {
		peer := peersBytes[i : i+6]
		var ipParts []string
		for _, b := range peer[:4] {
			ipParts = append(ipParts, strconv.Itoa(int(b)))
		}
		ip := strings.Join(ipParts, ".")
		port := binary.BigEndian.Uint16(peer[4:])
		peersList = append(peersList, PeerInfo{ip, port})
	}
	return peersList, nil
}

// See: https://wiki.theory.org/index.php/BitTorrentSpecification#Tracker_Request_Parameters
func peersRequestParams(tFile *File) (url.Values, error) {
	params := url.Values{}
	params.Add("info_hash", string(tFile.infoHash))
	peerId := [20]byte{'-', 'M', 'K', '0', '1', '0', '0', '-'}
	_, err := rand.Read(peerId[8:])
	log.Printf("peer id: %s%v\n", string(peerId[0:8]), peerId[8:])
	if err != nil {
		return nil, fmt.Errorf("failed to generate peer id: %v", err)
	}
	params.Add("peer_id", string(peerId[:]))
	params.Add("left", strconv.FormatInt(tFile.length(), 10))
	// TODO check if port is available, automatically select the other if not available
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("compact", "1")
	params.Add("no_peer_id", "true")
	params.Add("event", "started")
	return params, nil
}

func (f *File) length() int64 {
	if len(f.info.files) == 0 {
		return f.info.length
	}
	var res int64
	for _, file := range f.info.files {
		res += file.length
	}
	return res
}

func (f *File) PiecesNumber() int64 {
	return int64(len(f.info.pieces) / 20)
}
