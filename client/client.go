package client

import (
	"log"
	"math/rand"
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
	PieceIndex       int
	FixedPieceLength int
	Piece            []byte
}

type Client struct {
	bitField      []byte
	pieceNum      int
	wg            sync.WaitGroup
	metaInfo      *MetaInfo
	trackerClient *TrackerClient
	handShakeMsg  []byte
	downloadChan  chan DownloadPieceTask
	saveChan      chan SavePieceTask
	peerChan      chan *Peer
	downloadDir   string
	downloaderNum int
}

func NewClient(metaInfo *MetaInfo, downloadDir string, downloaderNum int) (*Client, error) {
	peerId := randomString(20)
	// peerId := "-UT0001-123456789012"
	peerPort := 6881

	trackerClient := NewTrackerClient(metaInfo.Announce, metaInfo.InfoHash, peerId,
		peerPort, 0, 0, metaInfo.Info.Length, 1, 50, "empty")

	return &Client{
		bitField:      GetBitfield(metaInfo, downloadDir, bitfieldDir),
		pieceNum:      len(metaInfo.Info.Pieces) / 20,
		metaInfo:      metaInfo,
		trackerClient: trackerClient,
		handShakeMsg:  handShakeMsg(metaInfo, peerId),
		downloadChan:  make(chan DownloadPieceTask, 100),
		saveChan:      make(chan SavePieceTask, 100),
		peerChan:      make(chan *Peer, 10),
		downloadDir:   downloadDir,
		downloaderNum: downloaderNum,
	}, nil
}

func (client *Client) StartDownload() {
	client.wg.Add(1)
	go client.SendDownloadTask()

	cancelChan := make(chan struct{})
	go client.FetchPeers(cancelChan)

	for i := 0; i < client.downloaderNum; i++ {
		client.wg.Add(1)
		go client.DownloadFromPeer(i)
	}

	client.wg.Add(1)
	go client.SavePiece()

	client.wg.Wait()
	close(cancelChan)
}

func (client *Client) SavePiece() {
	defer client.wg.Done()
	PieceSaver, err := NewPieceSaver(client.metaInfo, client.downloadDir, bitfieldDir)
	if err != nil {
		return
	}
	defer PieceSaver.Close()
	for saveTask := range client.saveChan {
		client.bitField[saveTask.PieceIndex/8] |= 1 << uint(7-saveTask.PieceIndex%8)
		err := PieceSaver.SavePiece(saveTask, client.bitField)
		if err != nil {
			log.Println("saving piece error ", err)
		} else {
			client.trackerClient.downloaded += len(saveTask.Piece)
			client.trackerClient.left -= len(saveTask.Piece)
			if client.trackerClient.left == 0 {
				client.trackerClient.event = "completed"
				log.Println("download finished")
				return
			}
		}
	}
}

func (client *Client) FetchPeers(cancelChan chan struct{}) {
	client.trackerClient.numwant = 50
	client.trackerClient.event = "started"
	for {
		select {
		case <-cancelChan:
			return
		default:
			res, err := client.trackerClient.Announce()
			if err != nil {
				panic(err)
			}
			for _, peer := range res.Peers {
				client.peerChan <- &peer
			}
		}
	}
}

func (client *Client) DownloadFromPeer(Id int) {
	defer client.wg.Done()
	for {
		downloader, err := NewDownloader(<-client.peerChan, client.handShakeMsg, client.bitField, Id)
		log.Println("new downloader ", Id)
		if err != nil {
			continue
		}
		err = downloader.Download(client.downloadChan, client.saveChan)
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
			// left some space of chan to avoid block
			if len(client.downloadChan) < cap(client.downloadChan)-client.downloaderNum {
				client.downloadChan <- DownloadPieceTask{i, pieceLength, client.metaInfo.Info.Pieces[i]}
			} else {
				time.Sleep(10 * time.Second)
				i--
			}
		}
	}

	close(client.downloadChan)
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
