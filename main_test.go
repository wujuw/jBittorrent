package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/wujuw/jBittorrent/client"
	"io"
	"log"
	"os"
	"strings"
	"testing"
)

func TestDownload(t *testing.T) {
	torrentPath := "download/ubuntu-22.10-desktop-amd64.iso.torrent"
	saveDir := "download"
	torrentFile, err := os.Open(torrentPath)
	if err != nil {
		t.Error("Error opening torrent file:", err)
	}
	defer torrentFile.Close()
	data, err := io.ReadAll(torrentFile)
	if err != nil {
		t.Error("Error reading torrent file:", err)
	}
	metaInfo, err := client.ParseMetaInfo(data)
	if err != nil {
		t.Error("Error parsing metainfo:", err)
	}
	c, err := client.NewClient(metaInfo, saveDir, 32)
	if err != nil {
		t.Error("Error creating peer client:", err)
	}

	c.StartDownload()

	downloadedFile, err := os.Open(saveDir + "/" + metaInfo.Info.Name)
	if err != nil {
		t.Error("Error opening downloaded file: ", err)
	}
	defer downloadedFile.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, downloadedFile); err != nil {
		t.Error(err)
	}
	sha256SUM := hex.EncodeToString(hash.Sum(nil))
	log.Println(fmt.Sprintf("下载文件的sha256为: %s", sha256SUM))

	sumsFile, err := os.Open(saveDir + "/SHA256SUMS")
	if err != nil {
		t.Error("Error opening sha256sums file: ", err)
	}
	defer sumsFile.Close()
	fileSums := make(map[string]string)
	scanner := bufio.NewScanner(sumsFile)
	for scanner.Scan() {
		line := scanner.Text()
		sums := strings.Split(line, " ")
		fileSums[strings.TrimPrefix(sums[1], "*")] = sums[0]
	}

	log.Println("文件原始sha256为: ", fileSums[metaInfo.Info.Name])
	if sha256SUM == fileSums[metaInfo.Info.Name] {
		log.Println("sha256一致，文件下载成功")
	} else {
		t.Error("sha256不一致，文件损坏")
	}
}
