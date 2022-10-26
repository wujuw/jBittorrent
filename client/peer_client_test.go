package client

import (
	"io"
	"os"
	"testing"
	// "time"
)

func TestHandShake(t *testing.T) {
	filePath := "/home/whhxd/codebase/jBittorrent/client/debian-11.5.0-amd64-DVD-1.iso.torrent" 
	file, err := os.Open(filePath)
	if err != nil {
		t.Errorf("could not open file: %s", filePath)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		t.Errorf("could not read file: %s, error: %s", filePath, err)
	}
	metaInfo, err := ParseMetaInfo(data)
	if err != nil {
		t.Error("could not parse meta info: ", err)
	}
	pc, err := NewPeerClient(metaInfo)
	if err != nil {
		t.Error("could not create peer client: ", err)
	}

	pc.Start()

	// sleep
	// time.Sleep(10 * time.Second)

}