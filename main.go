package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zeebo/bencode"
)

func main() {
	path := flag.String("torrent", "", "path to torrent file")
	flag.Parse()
	file, err := os.Open(*path)
	if err != nil {
		fmt.Printf("failed to open torrent file: %v\n", err)
		return
	}

	dec := bencode.NewDecoder(file)
	var torrent torrentFile
	if err := dec.Decode(&torrent); err != nil {
		fmt.Printf("failed to decode torrent file: %v\n", err)
		return
	}
	fmt.Printf("%+v\n", torrent)
}

type torrentFile struct {
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
