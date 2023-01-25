package torrent

import (
	"crypto/sha1"
	"fmt"
	"github.com/zeebo/bencode"
	"io"
	"os"
	"strings"
)

// File contains metainfo from the torrent file.
// See: https://wiki.theory.org/index.php/BitTorrentSpecification#Metainfo_File_Structure
type File struct {
	announce string
	info     struct {
		pieces string
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
	WarningMessage string
	Interval       int64
	MinInterval    int64
	TrackerId      string
	Complete       int64
	Incomplete     int64
	Peers          []PeerInfo
}

type PeerInfo struct {
	IP   string
	Port uint16
}

// Parse extracts a metainfo from the torrent file specified by path argument.
func Parse(path string) (*File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open torrent file: %v\n", err)
	}
	var metainfoMap map[string]interface{}
	d := bencode.NewDecoder(file)
	if err := d.Decode(&metainfoMap); err != nil {
		return nil, fmt.Errorf("failed to decode torrent file: %v\n", err)
	}
	var tFile File
	announce, ok := metainfoMap["announce"]
	if !ok {
		return nil, fmt.Errorf("cannot find 'announce' field in the torrent file")
	}
	tFile.announce = announce.(string)
	info, ok := metainfoMap["info"]
	if !ok {
		return nil, fmt.Errorf("cannot find 'info' field in the torrent file")
	}
	infoMap := info.(map[string]interface{})
	hash, err := infoHash(infoMap)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate 'info' hash")
	}
	tFile.infoHash = hash
	if pieces, ok := infoMap["pieces"]; ok {
		tFile.info.pieces = pieces.(string)
	}
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
