package main

import (
	"fmt"
	"io"
	"os"
	"github.com/wujuw/jBittorrent/client"
	// "time"
)

func main() {
	if (len(os.Args) != 2) {
		fmt.Println("Usage: ", os.Args[0], " <torrent file>")
		os.Exit(1)
	}
	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println("Error opening file: ", err)
		os.Exit(1)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file: ", err)
		os.Exit(1)
	}
	metaInfo, err := client.ParseMetaInfo(data)
	if err != nil {
		fmt.Println("Error parsing metainfo: ", err)
		os.Exit(1)
	}
	pc, err := client.NewPeerClient(metaInfo)
	if err != nil {
		fmt.Println("Error creating peer client: ", err)
		os.Exit(1)
	}

	err = pc.Start()
	if err != nil {
		fmt.Println("Error starting peer client: ", err)
		os.Exit(1)
	}

	// time.Sleep(100 * time.Second)
}