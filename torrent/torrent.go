package torrent

import (
	"fmt"
	"github.com/zeebo/bencode"
	"os"
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
