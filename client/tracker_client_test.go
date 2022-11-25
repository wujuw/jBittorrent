package client

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func TestTrackerClient(t *testing.T) {
	file, err := os.Open("../download/ubuntu-22.10-desktop-amd64.iso.torrent")
	if err != nil {
		t.Error("Error opening file: ", err)
	}
	defer file.Close()
	// Read the file into a byte array
	data := make([]byte, 1024*1024)
	n, err := file.Read(data)
	if err != nil && err != io.EOF {
		t.Error("Error reading file: ", err)
	}
	var metaInfo *MetaInfo
	metaInfo, err = ParseMetaInfo(data[:n])
	if err != nil {
		t.Error("Error parsing metainfo: ", err)
	}
	peerId := "-JB0001-123456789012"
	fmt.Println(metaInfo.InfoHash)
	trackerClient := NewTrackerClient(metaInfo.Announce, metaInfo.InfoHash, peerId, 6881, 0, 0, metaInfo.Info.Length, 1, 4, "empty")
	_, err = trackerClient.Announce()
	if err != nil {
		t.Error("Error announcing to tracker: ", err)
	}
	_, err = trackerClient.AnnounceWithoutCompact() //解析无压缩的Peers
	if err != nil {
		t.Error("Error announcing to tracker: ", err)
	}
}
