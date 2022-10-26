package client

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net"
	"time"
)

const (
	pstrlen = 19
	pstr = "BitTorrent protocol"
)

var reserved = [8]byte{0, 0, 0, 0, 0, 0, 0, 0}

type PeerClient struct {
	peerId string
	peerPort int
	metaInfo *MetaInfo
	trackerClient *TrackerClient
	handShakeMsg []byte
}

func NewPeerClient(metaInfo *MetaInfo) (*PeerClient, error) {
	peerId := randomString(20)
	// peerId := "-UT0001-123456789012"
	peerPort := 6881

	for peerPort < 6889 {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", peerPort))
		if err != nil {
			peerPort++
		} else {
			ln.Close()
			break
		}
	}
	if peerPort == 6889 {
		return nil, fmt.Errorf("could not find a free port")
	}

	trackerClient := NewTrackerClient(metaInfo.Announce, metaInfo.InfoHash, peerId,
		 peerPort, 0, 0, metaInfo.Info.Length, 1, 50, "empty")

	handShakeMsg := make([]byte, 68)
	handShakeMsg[0] = byte(pstrlen)
	copy(handShakeMsg[1:20], []byte(pstr))
	copy(handShakeMsg[20:28], reserved[:])
	copy(handShakeMsg[28:48], []byte(metaInfo.InfoHash)[:])
	copy(handShakeMsg[48:68], []byte(peerId))

	return &PeerClient{
		peerId: peerId,
		peerPort: peerPort,
		metaInfo: metaInfo,
		trackerClient: trackerClient,
		handShakeMsg: handShakeMsg,
	}, nil
}

func (client *PeerClient) Start() error {
	trackerResponse, err := client.trackerClient.Announce()
	if err != nil {
		return err
	}
	
	for _, peer := range trackerResponse.Peers {

		
			conn, err := client.Connect(&peer)
			if err != nil {
				fmt.Println("Error connecting to peer: ", err)
				continue
			}

			client.HandShake(&peer, conn)
		
	}
	return nil
}

func (client *PeerClient) Connect(server *Peer) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", server.IP, server.Port), 2 * time.Second)
	// conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", server.IP, server.Port))

	if err != nil {
		fmt.Println("Error connecting to peer: ", err)
		return nil, err
	}

	return conn, nil
}

func (client *PeerClient) HandShake(server *Peer, conn net.Conn) error {
	_, err := conn.Write(client.handShakeMsg)
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

	if !bytes.Equal(resp[0:20], client.handShakeMsg[0:20]) ||
		!bytes.Equal(resp[28:48], client.handShakeMsg[28:48]) || 
		(server.PeerId != "" && !bytes.Equal(resp[48:68], []byte(server.PeerId))) {
		return fmt.Errorf("handshake response: %s is not valid", resp)
	}

	fmt.Println("Handshake successful")

	return nil
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