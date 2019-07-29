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
	var meta metaInfo
	if err := dec.Decode(&meta); err != nil {
		fmt.Printf("failed to decode torrent file: %v\n", err)
		return
	}
	fmt.Printf("%+v\n", meta)
}

type metaInfo struct {
	Announce string
	Info     struct {
		PieceLen int64 `bencode:"piece length"`
		Pieces   string
		Length   int64
		Files    []struct {
			Length int64
			Path   []string
		}
	}
}
