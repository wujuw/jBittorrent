package client

// import (
// 	"testing"
// 	"os"
// 	"fmt"
// 	"io"
// )

// func TestMain(t *testing.T) {
// 	os.Args[1] = "debian-11.5.0-amd64-DVD-1.iso.torrent"
// 	os.Args[2] = "/home/whhxd/codebase/jBittorrent/client/download"
// 	file, err := os.Open(os.Args[1])
// 	if err != nil {
// 		fmt.Println("Error opening file:", err)
// 		os.Exit(1)
// 	}
// 	data, err := io.ReadAll(file)
// 	if err != nil {
// 		fmt.Println("Error reading file:", err)
// 		os.Exit(1)
// 	}
// 	metaInfo, err := ParseMetaInfo(data)
// 	if err != nil {
// 		fmt.Println("Error parsing metainfo:", err)
// 		os.Exit(1)
// 	}
// 	client, err := NewClient(metaInfo, os.Args[2])
// 	if err != nil {
// 		fmt.Println("Error creating peer client:", err)
// 		os.Exit(1)
// 	}

// 	client.StartDownload()
// }