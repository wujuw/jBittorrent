package client

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	pstrlen     = 19
	pstr        = "BitTorrent protocol"
	bitfieldDir = "bitfield"
)

var reserved = [8]byte{0, 0, 0, 0, 0, 0, 0, 0}

type DownloadPieceTask struct {
	PieceIndex  int
	PieceLength int
	PieceHash   [20]byte
}

type SavePieceTask struct {
	PieceIndex int
	Piece      []byte
}

type Client struct {
	bitField      []byte
	pieceNum      int
	savedNum      int
	wg            sync.WaitGroup
	metaInfo      *MetaInfo
	handShakeMsg  []byte
	downloadChan  chan DownloadPieceTask
	saveChan      chan SavePieceTask
	fallbackChan  chan DownloadPieceTask
	peerChan      chan *Peer
	downloadDir   string
	downloaderNum int
	peerId        string
	peerPort      int
	peers         map[int]*Peer
	paused        bool
	speed         string
	cancelChan    chan struct{}
}

func NewClient(metaInfo *MetaInfo, downloadDir string, downloaderNum int) (*Client, error) {
	peerId := randomString(20)
	// peerId := "-UT0001-123456789012"
	peerPort := 6881

	bitfield := GetBitfield(metaInfo, downloadDir, bitfieldDir)

	return &Client{
		bitField:      bitfield,
		pieceNum:      len(metaInfo.Info.Pieces),
		metaInfo:      metaInfo,
		handShakeMsg:  handShakeMsg(metaInfo, peerId),
		downloadChan:  make(chan DownloadPieceTask, 100),
		fallbackChan:  make(chan DownloadPieceTask, downloaderNum+1),
		saveChan:      make(chan SavePieceTask, 100),
		peerChan:      make(chan *Peer, downloaderNum),
		downloadDir:   downloadDir,
		downloaderNum: downloaderNum,
		savedNum:      calcSavedNum(bitfield, len(metaInfo.Info.Pieces)),
		peerId:        peerId,
		peerPort:      peerPort,
		peers:         make(map[int]*Peer, downloaderNum),
		paused:        true,
		speed:         "0B/S",
		cancelChan:    make(chan struct{}),
	}, nil
}

func (client *Client) StartDownload() {
	client.paused = false
	client.wg.Add(1)
	go client.SendDownloadTask()

	go client.FetchPeers(client.cancelChan)

	for i := 0; i < client.downloaderNum; i++ {
		go client.DownloadFromPeer(i)
	}

	client.wg.Add(1)
	go client.SavePiece()

	go client.calcSpeed()
	client.wg.Wait()
	//close(client.cancelChan)
	client.paused = true
}

func (client *Client) SavePiece() {
	defer client.wg.Done()
	PieceSaver, err := NewPieceSaver(client.metaInfo, client.downloadDir, bitfieldDir)
	if err != nil {
		return
	}
	defer PieceSaver.Close()
	var saveTask SavePieceTask
	if client.savedNum != client.pieceNum {
		for saveTask = range client.saveChan {
			client.bitField[saveTask.PieceIndex/8] |= 1 << uint(7-saveTask.PieceIndex%8)
			err := PieceSaver.SavePiece(saveTask, client.bitField)
			if err != nil {
				log.Println("saving piece error ", err)
				panic(err)
			} else {
				client.savedNum++
				if client.savedNum == client.pieceNum {
					close(client.saveChan)
				}
			}
		}
	}
	if client.savedNum == client.pieceNum {
		log.Println("download finished")
		close(client.fallbackChan)
		close(client.downloadChan)
	} else {
		log.Println("download stop")
		close(client.fallbackChan)
	}
	return
}

func (client *Client) FetchPeers(cancelChan chan struct{}) {
	trackerList := make([]string, 0, 50)
	trackerList = append(trackerList, client.metaInfo.Announce)
	for _, urlList := range client.metaInfo.AnnounceList {
		for _, trackerUrl := range urlList {
			// 只支持http tracker
			if strings.HasPrefix(trackerUrl, "http") {
				trackerList = append(trackerList, trackerUrl)
			}
		}
	}

	for _, trackerUrl := range trackerList {
		select {
		case <-cancelChan:
			return
		default:
			go client.FetchPeersFromTracker(trackerUrl)
		}
	}
}

func (client *Client) FetchPeersFromTracker(trackerUrl string) {
	trackerClient := NewTrackerClient(trackerUrl, client.metaInfo.InfoHash, client.peerId,
		client.peerPort, 0, 0, client.metaInfo.Info.Length, 1, 50, "started")
	res, err := trackerClient.Announce()
	if err != nil {
		log.Println("warning: request " + trackerUrl + " failed, error: " + err.Error())
		return
	}
	for i, _ := range res.Peers {
		client.peerChan <- &res.Peers[i]
	}
}

func (client *Client) DownloadFromPeer(Id int) {
	for {
		peer := <-client.peerChan
		downloader, err := NewDownloader(peer, client.handShakeMsg, client.bitField, Id)
		log.Println("new downloader ", Id)
		if err != nil {
			continue
		}
		client.peers[Id] = peer
		client.wg.Add(1)
		err = downloader.Download(client.downloadChan, client.saveChan, client.fallbackChan, client.cancelChan)
		client.wg.Done()
		delete(client.peers, Id)
		if err != nil {
			log.Println("downloader error ", err)
			continue
		} else {
			break
		}
	}
}

func (client *Client) SendDownloadTask() {
	defer client.wg.Done()

	for i := 0; i < client.pieceNum; i++ {
		bitFiledIndex := i / 8
		bitFiledOffset := i % 8
		if client.bitField[bitFiledIndex]&(1<<uint(7-bitFiledOffset)) == 0 {
			var pieceLength int
			if i == client.pieceNum-1 {
				pieceLength = client.metaInfo.Info.Length % client.metaInfo.Info.PieceLength
			} else {
				pieceLength = client.metaInfo.Info.PieceLength
			}
			for len(client.fallbackChan) > 0 {
				client.downloadChan <- <-client.fallbackChan
			}
			client.downloadChan <- DownloadPieceTask{i, pieceLength, client.metaInfo.Info.Pieces[i]}
		}
	}

	for failTask := range client.fallbackChan {
		client.downloadChan <- failTask
	}
}

func (client *Client) GetDownloadProcess() map[string]string {
	info := make(map[string]string)
	downloadedBytes := client.savedNum * client.metaInfo.Info.PieceLength
	if downloadedBytes < 1024 {
		info["downloaded"] = strconv.Itoa(downloadedBytes) + "B"
	} else if downloadedBytes < 1024*1024 {
		info["downloaded"] = strconv.Itoa(downloadedBytes/1024) + "KB"
	} else if downloadedBytes < 1024*1024*1024 {
		info["downloaded"] = strconv.Itoa(downloadedBytes/(1024*1024)) + "MB"
	} else {
		info["downloaded"] = fmt.Sprintf("%.1f", float64(downloadedBytes)/float64(1024*1024*1024)) + "GB"
	}
	all := client.metaInfo.Info.Length
	if all < 1024 {
		info["all"] = strconv.Itoa(all) + "B"
	} else if all < 1024*1024 {
		info["all"] = strconv.Itoa(all/1024) + "KB"
	} else if all < 1024*1024*1024 {
		info["all"] = strconv.Itoa(all/(1024*1024)) + "MB"
	} else {
		info["all"] = fmt.Sprintf("%.1f", float64(all)/float64(1024*1024*1024)) + "GB"
	}
	percent := float64(downloadedBytes) / float64(all) * 100
	if percent-100 > 0 {
		percent = 100
	}
	info["percent"] = fmt.Sprintf("%.2f", percent) + "%"
	info["speed"] = client.speed

	return info
}

func (client *Client) GetPeers() map[int]*Peer {
	return client.peers
}

func (client *Client) Stop() {
	close(client.cancelChan)
	time.Sleep(time.Second * 3)
}

func (client *Client) calcSpeed() {
	for {
		start := time.Now().Second()
		startNum := client.savedNum
		time.Sleep(time.Second * 1)
		end := time.Now().Second()
		endNum := client.savedNum

		speed := (endNum - startNum) * client.metaInfo.Info.PieceLength / (end - start) // Bytes/s
		speedStr := ""
		if speed < 1024 {
			speedStr = strconv.Itoa(speed) + "B/S"
		} else if speed < 1024*1024 {
			speedStr = strconv.Itoa(speed/1024) + "KB/S"
		} else if speed < 1024*1024*1024 {
			speedStr = fmt.Sprintf("%.1f", float64(speed)/float64(1024*1024)) + "MB/S"
		} else {
			speedStr = fmt.Sprintf("%.1f", float64(speed)/float64(1024*1024*1024)) + "GB/S"
		}
		client.speed = speedStr
	}
}

func handShakeMsg(metaInfo *MetaInfo, clientId string) []byte {
	msg := make([]byte, 68)
	msg[0] = byte(pstrlen)
	copy(msg[1:20], []byte(pstr))
	copy(msg[20:28], reserved[:])
	copy(msg[28:48], []byte(metaInfo.InfoHash)[:])
	copy(msg[48:68], []byte(clientId))
	return msg
}

var defaultLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// RandomString returns a random string with a fixed length
func randomString(n int, allowedChars ...[]rune) string {
	var letters []rune

	if len(allowedChars) == 0 {
		letters = defaultLetters
	} else {
		letters = allowedChars[0]
	}

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

func calcSavedNum(bitField []byte, pieceNum int) int {
	savedNum := 0
	for i := 0; i < pieceNum; i++ {
		bitFiledIndex := i / 8
		bitFiledOffset := i % 8
		if bitField[bitFiledIndex]&(1<<uint(7-bitFiledOffset)) == 1<<uint(7-bitFiledOffset) {
			savedNum++
		}
	}
	return savedNum
}
