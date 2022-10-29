package client

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type Downloader struct {
	bitfield []byte
	conn     net.Conn
	state    *State
	Id       int
}

type State struct {
	am_choking      bool
	am_interested   bool
	peer_choking    bool
	peer_interested bool
}

func NewDownloader(peer *Peer, handShakeMsg []byte, bitfield []byte, Id int) (*Downloader, error) {
	conn, err := Connect(peer)
	if err != nil {
		fmt.Println("Error connecting to peer: ", err)
		return nil, err
	}
	err = HandShake(peer, handShakeMsg, conn)
	if err != nil {
		fmt.Println("Error handshaking with peer: ", err)
		return nil, err
	}

	state := &State{
		am_choking:      true,
		am_interested:   false,
		peer_choking:    true,
		peer_interested: false,
	}

	sendBitfield(conn, bitfield)

	bitfieldMsg, err := getBitfield(conn, state)
	if err != nil {
		fmt.Println("Error getting bitfield from peer: ", err)
		return nil, err
	}

	if err != nil {
		fmt.Println("Error sending interested to peer: ", err)
		return nil, err
	}

	return &Downloader{
		bitfield: bitfieldMsg.payload,
		conn:     conn,
		state:    state,
		Id:       Id,
	}, nil
}

func Connect(server *Peer) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("[%s]:%d", server.IP, server.Port), 2*time.Second)
	// conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", server.IP, server.Port))

	if err != nil {
		fmt.Println("Error connecting to peer: ", err)
		return nil, err
	}

	return conn, nil
}

func HandShake(server *Peer, handShakeMsg []byte, conn net.Conn) error {
	_, err := conn.Write(handShakeMsg)
	if err != nil {
		fmt.Println("Error writing handshake: ", err)
		return fmt.Errorf("could not send handshake message: %s", err)
	}

	resp := make([]byte, 68)
	n, err := io.ReadFull(conn, resp)
	if err != nil {
		fmt.Println("Error reading handshake: ", err)
		return err
	}
	if n != 68 {
		return fmt.Errorf("handshake response is not 68 bytes")
	}

	if !bytes.Equal(resp[0:20], handShakeMsg[0:20]) ||
		!bytes.Equal(resp[28:48], handShakeMsg[28:48]) ||
		(server.PeerId != "" && !bytes.Equal(resp[48:68], []byte(server.PeerId))) {
		//return fmt.Errorf("handshake response: %s is not valid", resp)
	}

	fmt.Println("Handshake successful")

	return nil
}

func (client *Downloader) Download(downloadChan <-chan DownloadPieceTask, saveChan chan SavePieceTask,
	fallbackChan chan DownloadPieceTask) error {
	defer client.conn.Close()
	go client.Keepalive()

	for task := range downloadChan {
		fmt.Println("Downloading piece: ", task.PieceIndex)
		if (client.bitfield[task.PieceIndex/8] & (1 << uint(7-(task.PieceIndex%8)))) == 0 {
			fmt.Println("Peer does not have piece: ", task)
			fallbackChan <- task
			continue
		}
		sendInterested(client.conn)
		client.state.am_interested = true
		for client.state.peer_choking {
			log.Default().Printf("Downloader %d is choking, waiting for unchoke", client.Id)
			msg, err := ReadMessageFrom(client.conn)
			if err != nil {
				fmt.Println("Error reading message: ", err)
				fallbackChan <- task
				return err
			}
			switch msg.typeId {
			case Unchoke:
				client.state.peer_choking = false
			case Choke:
				client.state.peer_choking = true
			case Have:
				index := int(msg.payload[0])<<24 | int(msg.payload[1])<<16 | int(msg.payload[2])<<8 | int(msg.payload[3])
				client.bitfield[index/8] |= 1 << uint(7-(index%8))
			}
		}
		fmt.Println("Starting download of piece: ", task.PieceIndex)
		piece := make([]byte, task.PieceLength)
		slicebegin := 0
		slicelength := 16384

		sliceNum := task.PieceLength/slicelength + 1

		for slicebegin < task.PieceLength {
			slicebeginSend := slicebegin
			slicelengthSend := slicelength
			for i := 0; i < sliceNum; i++ {
				if slicebeginSend < task.PieceLength {
					if slicebeginSend+slicelengthSend > task.PieceLength {
						slicelengthSend = task.PieceLength - slicebeginSend
					}
					err := sendRequest(client.conn, task.PieceIndex, slicebeginSend, slicelengthSend)
					if err != nil {
						fmt.Println("Error sending request: ", err)
						fallbackChan <- task
						return err
					}
					slicebeginSend += slicelengthSend
				} else {
					break
				}
			}
			for i := 0; i < sliceNum; i++ {
				if slicebegin < task.PieceLength {
					for pieceMsg := false; !pieceMsg; {
						msg, err := ReadMessageFrom(client.conn)
						if err != nil {
							fmt.Println("Error reading message: ", err)
							fallbackChan <- task
							return err
						}
						switch msg.typeId {
						case Piece:
							pieceIndex := uint32(BytesToInt32(msg.payload[0:4]))
							begin := uint32(BytesToInt32(msg.payload[4:8]))
							slice := msg.payload[8:]
							if pieceIndex != uint32(task.PieceIndex) {
								fmt.Println("Error: piece index does not match")
								continue
							} else if begin != uint32(slicebegin) {
								fmt.Println("Error: begin does not match")
								continue
							}
							copy(piece[slicebegin:], slice)
							slicebegin += len(slice)
							fmt.Printf("Downloaded slice of piece %d, slice begin:%d, slice length: %dB\n", task.PieceIndex, slicebegin, slicelength)
							pieceMsg = true
						}
					}
				} else {
					break
				}
			}
		}
		downloadPieceHash := sha1.Sum(piece)
		if !bytes.Equal(downloadPieceHash[:], task.PieceHash[:]) {
			fmt.Println("Error: piece hash does not match")
			fallbackChan <- task
			continue
		}
		fmt.Println("Downloaded piece: ", task.PieceIndex)
		saveChan <- SavePieceTask{PieceIndex: task.PieceIndex, Piece: piece}
	}
	log.Printf("downloader %d work done.\n", client.Id)
	return nil
}

func (downloader *Downloader) Keepalive() error {
	for {
		time.Sleep(30 * time.Second)
		err := SendKeepalive(downloader.conn)
		if err != nil {
			log.Default().Printf("Error sending keepalive: %s", err)
			return err
		}
	}
}

func sendBitfield(conn net.Conn, bitfield []byte) error {
	msg := Message{
		typeId:  Bitfield,
		payload: bitfield,
	}
	_, err := msg.WriteTo(conn)
	return err
}

func getBitfield(conn net.Conn, state *State) (*Message, error) {
	for {
		bitfieldMsg, err := ReadMessageFrom(conn)
		if err != nil {
			return nil, err
		}
		switch bitfieldMsg.typeId {
		case Bitfield:
			return bitfieldMsg, nil
		case Unchoke:
			state.peer_choking = false
		default:
			continue
		}
	}
}

// func sendBitfield(conn net.Conn) error {
// 	bitfieldMsg := NewMessage(Bitfield, client.bitfield)
// 	_, err := bitfieldMsg.WriteTo(conn)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func sendInterested(conn net.Conn) error {
	interestedMsg := NewMessage(Interested, nil)
	_, err := interestedMsg.WriteTo(conn)
	if err != nil {
		return err
	}
	return nil
}

// func sendNotInterested(conn net.Conn) error {
// 	notInterestedMsg := NewMessage(NotInterested, nil)
// 	_, err := notInterestedMsg.WriteTo(conn)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func sendChoke(conn net.Conn) error {
// 	chokeMsg := NewMessage(Choke, nil)
// 	_, err := chokeMsg.WriteTo(conn)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func sendUnchoke(conn net.Conn) error {
// 	unchokeMsg := NewMessage(Unchoke, nil)
// 	_, err := unchokeMsg.WriteTo(conn)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func sendRequest(conn net.Conn, index, begin, length int) error {
	requestMsg := NewRequestMessage(index, begin, length)
	_, err := requestMsg.WriteTo(conn)
	if err != nil {
		return err
	}
	return nil
}

// func sendCancel(conn net.Conn, index, begin, length int) error {
// 	cancelMsg := NewCancelMessage(index, begin, length)
// 	_, err := cancelMsg.WriteTo(conn)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
