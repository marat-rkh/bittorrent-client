package torrent

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
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

// File contains metainfo from the torrent file.
// See: https://wiki.theory.org/index.php/BitTorrentSpecification#Metainfo_File_Structure
type File struct {
	announce string
	info     struct {
		length int64
		files  []fileInfo
	}
	infoHash []byte
}

type fileInfo struct {
	length int64
	path   []string
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
	Peers          string `bencode:"peers"`
}

// Parse extracts a metainfo from the torrent file specified by path argument.
func Parse(path string) (*File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open torrent file: %v\n", err)
	}
	var decoded map[string]interface{}
	d := bencode.NewDecoder(file)
	if err := d.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("failed to decode torrent file: %v\n", err)
	}
	var tFile File
	announce, ok := decoded["announce"]
	if !ok {
		return nil, fmt.Errorf("cannot find 'announce' field in the torrent file")
	}
	tFile.announce = announce.(string)
	info, ok := decoded["info"]
	if !ok {
		return nil, fmt.Errorf("cannot find 'info' field in the torrent file")
	}
	infoMap := info.(map[string]interface{})
	hash, err := infoHash(infoMap)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate 'info' hash")
	}
	tFile.infoHash = hash
	if length, ok := infoMap["length"]; ok {
		tFile.info.length = length.(int64)
	} else if files, ok := infoMap["files"]; ok {
		filesList := files.([]interface{})
		for _, file := range filesList {
			fileMap := file.(map[string]interface{})
			fLen, ok := fileMap["length"]
			if !ok {
				return nil, fmt.Errorf("cannot find 'info.files.length' field in the torrent file")
			}
			fPath, ok := fileMap["path"]
			if !ok {
				return nil, fmt.Errorf("cannot find 'info.files.path' field in the torrent file")
			}
			tFile.info.files = append(tFile.info.files, fileInfo{
				length: fLen.(int64),
				path:   asStringSlice(fPath.([]interface{})),
			})
		}
	} else {
		return nil, fmt.Errorf("cannot find 'info.length' or 'info.files' field in the torrent file")
	}
	return &tFile, nil
}

func infoHash(decodedInfoMap interface{}) ([]byte, error) {
	info := strings.Builder{}
	enc := bencode.NewEncoder(&info)
	err := enc.Encode(decodedInfoMap)
	if err != nil {
		return nil, fmt.Errorf("failed to encode torrent info: %v", err)
	}
	hash := sha1.New()
	_, _ = io.WriteString(hash, info.String())
	return hash.Sum(nil), nil
}

func asStringSlice(is []interface{}) []string {
	var ss []string
	for _, i := range is {
		ss = append(ss, i.(string))
	}
	return ss
}

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
	log.Printf("responce body: \"%s\"", body.String())
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
