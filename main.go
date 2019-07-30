package main

import (
	"bittorrent-client/torrent"
	"flag"
	"fmt"
)

func main() {
	path := flag.String("torrent", "", "path to torrent file")
	flag.Parse()
	tFile, err := torrent.Parse(*path)
	if err != nil {
		fmt.Printf("failed to parse torrent file: %v\n", err)
		return
	}
	resp, err := tFile.MakeAnnounceRequest()
	if err != nil {
		fmt.Printf("announce request failed: %v\n", err)
		return
	}
	fmt.Printf("%+v\n", resp)
}
