package client

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"io"
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
		 peerPort, 0, 0, metaInfo.Info.Length, 1, 10, "empty")

	handShakeMsg := make([]byte, 68)
	handShakeMsg[0] = byte(pstrlen)
	copy(handShakeMsg[1:20], []byte(pstr))
	copy(handShakeMsg[20:28], reserved[:])
	copy(handShakeMsg[28:48], metaInfo.InfoHash[:])
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
	defer client.trackerClient.httpClient.CloseIdleConnections()
	conn, err := client.Connect(&trackerResponse.Peers[0])
	if err != nil {
		return err
	}
	defer conn.Close()

	err = client.HandShake(&trackerResponse.Peers[0], conn)
	if err != nil {
		return err
	}
	return nil
}

func (client *PeerClient) Connect(server *Peer) (*net.TCPConn, error) {
	lAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", client.peerPort))
	if err != nil {
		return nil, err
	}
	rAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", server.IP, server.Port))
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", lAddr, rAddr)
	if err != nil {
		return nil, err
	}
	
	return conn, nil
}

func (client *PeerClient) HandShake(server *Peer, conn *net.TCPConn) error {
	_, err := conn.Write(client.handShakeMsg)
	if err != nil {
		return fmt.Errorf("could not send handshake message: %s", err)
	}

	resp := make([]byte, 68)
	n, err := io.ReadFull(conn, resp)
	if err != nil {
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