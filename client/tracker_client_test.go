package client

import (
	"testing"
	"fmt"
	"os"
	"io"
)

func TestTrackerClient(t *testing.T) {
	file, err := os.Open("debian-11.5.0-amd64-DVD-1.iso.torrent")
	if err != nil {
		t.Error("Error opening file: ", err)
	}
	defer file.Close()
	// Read the file into a byte array
	data := make([]byte, 1024 * 1024)
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
	trackerClient := NewTrackerClient(metaInfo.Announce, metaInfo.InfoHash, peerId, 6881, 0, 0, metaInfo.Info.Length, 0, 4, "empty")
	trackerResponse, err := trackerClient.Announce()
	if err != nil {
		t.Error("Error announcing to tracker: ", err)
	}
	fmt.Print(trackerResponse)
}
