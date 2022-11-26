package main

import (
	"fmt"
	"github.com/wujuw/jBittorrent/client"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage:", os.Args[0], " <torrent file>", "<destination directory>")
		os.Exit(1)
	}
	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
	}
	metaInfo, err := client.ParseMetaInfo(data)
	if err != nil {
		fmt.Println("Error parsing metainfo:", err)
		os.Exit(1)
	}
	c, err := client.NewClient(metaInfo, os.Args[2], 64)
	if err != nil {
		fmt.Println("Error creating peer client:", err)
		os.Exit(1)
	}

	go c.StartDownload()
	log.SetOutput(io.Discard)

	var command string
	for {
		fmt.Println("Please input command: ")
		fmt.Scan(&command)
		switch command {
		case "process":
			info := c.GetDownloadProcess()
			fmt.Println(fmt.Sprintf("downloaded: %s / %s, %s, download speed: %s", info["downloaded"], info["all"], info["percent"], info["speed"]))
		case "peers":
			peers := c.GetPeers()
			keys := make([]int, 0, len(peers))
			for key := range peers {
				keys = append(keys, key)
			}
			sort.Ints(keys)
			for _, id := range keys {
				fmt.Println(fmt.Sprintf("downloader %s connected peer: [%s]:%s", strconv.Itoa(id), peers[id].IP, strconv.Itoa(peers[id].Port)))
			}
		case "exit":
			c.Stop()
			os.Exit(0)
		default:
			fmt.Println("support command: ")
			fmt.Println("process: show the download process")
			fmt.Println("peers: show the connected peers")
			fmt.Println("exit: stop the download")
		}
	}
}
