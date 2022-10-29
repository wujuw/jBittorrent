package client

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Message struct {
	typeId  byte
	payload []byte
}

const (
	Choke         = 0
	Unchoke       = 1
	Interested    = 2
	NotInterested = 3
	Have          = 4
	Bitfield      = 5
	Request       = 6
	Piece         = 7
	Cancel        = 8
	Keepalive     = 9
)

func NewMessage(typeId byte, payload []byte) *Message {
	return &Message{
		typeId:  typeId,
		payload: payload,
	}
}

func NewRequestMessage(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return NewMessage(Request, payload)
}

func NewCancelMessage(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return NewMessage(Cancel, payload)
}

func BytesToInt32(bytes []byte) uint32 {
	return binary.BigEndian.Uint32(bytes)
}

func ReadMessageFrom(r io.Reader) (*Message, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	if length == 0 {
		return &Message{typeId: Keepalive}, nil
	}

	data := make([]byte, length)
	if err := binary.Read(r, binary.BigEndian, &data); err != nil {
		return nil, err
	}

	if length == 1 {
		return NewMessage(data[0], nil), nil
	} else {
		fmt.Println("length: ", length)
		return NewMessage(data[0], data[1:]), nil
	}
}

func (m *Message) WriteTo(w io.Writer) (int64, error) {
	if err := binary.Write(w, binary.BigEndian, uint32(len(m.payload)+1)); err != nil {
		return 0, err
	}

	if err := binary.Write(w, binary.BigEndian, m.typeId); err != nil {
		return 0, err
	}

	if err := binary.Write(w, binary.BigEndian, m.payload); err != nil {
		return 0, err
	}

	return int64(4 + len(m.payload) + 1), nil
}

func SendKeepalive(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, uint32(0))
}
