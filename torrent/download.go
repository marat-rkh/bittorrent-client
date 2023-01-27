package torrent

type Download struct {
	torrent     *File
	trackerResp *TrackerResponse
	status      *status
}

func NewDownload(torrent *File, resp *TrackerResponse) *Download {
	return &Download{torrent: torrent, trackerResp: resp}
}

func (d *Download) Start() {
	for i := range d.trackerResp.Peers {
		s := session{download: d, peerIdx: i, piecesQueue: make(chan int)}
		go func() {
			_ = s.start()
		}()
	}
}
