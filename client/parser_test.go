package client

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestReadLengthPrefix(t *testing.T) {
	data := []byte("4:spam")
	lengthPrefix, readLen, err := readLengthPrefix(data)
	if err != nil {
		t.Error("Error reading length prefix: ", err)
	}
	if lengthPrefix != 4 {
		t.Error("Expected length to be 4, got ", lengthPrefix)
	}
	if readLen != 2 {
		t.Error("Expected byteLen to be 1, got ", readLen)
	}
}

func TestReadString(t *testing.T) {
	data := []byte("4:spam")
	str, readLen, err := readString(data)
	if err != nil {
		t.Error("Error reading string: ", err)
	}
	if str != "spam" {
		t.Error("Expected string to be 'spam', got ", str)
	}
	if readLen != 6 {
		t.Error("Expected byteLen to be 5, got ", readLen)
	}
}

func TestReadInt(t *testing.T) {
	dataList := [][]byte{[]byte("i3e"), []byte("i-31e"), []byte("i31e"), []byte("i0e")}
	expected := [][2]int{{3, 3}, {-31, 5}, {31, 4}, {0, 3}}
	for i, data := range dataList {
		intVal, readLen, err := readInt(data)
		if err != nil {
			t.Error("Error reading int: ", err)
		}
		if intVal != expected[i][0] {
			t.Errorf("Expected int value to be %d, got %d", expected[i][0], intVal)
		}
		if readLen != int(expected[i][1]) {
			t.Errorf("Expected readLen to be %d, got %d", expected[i][1], readLen)
		}
	}
}

func TestParseMetaInfo(t *testing.T) {
	data := []byte("d8:announce35:http://tracker.example.com/announce13:announce-listll35:http://tracker.example.com/announceel36:http://tracker2.example.com/announceee4:infod6:lengthi123456e4:name4:spam12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaaee")
	metaInfo, err := ParseMetaInfo(data)
	if err != nil {
		t.Error("Error parsing metainfo: ", err)
	}
	if metaInfo.Announce != "http://tracker.example.com/announce" {
		t.Error("Expected announce to be 'http://tracker.example.com/announce', got ", metaInfo.Announce)
	}
	if metaInfo.AnnounceList[0][0] != "http://tracker.example.com/announce" {
		t.Error("Expected announce-list to be 'http://tracker.example.com/announce', got ", metaInfo.AnnounceList[0][0])
	}
	if metaInfo.AnnounceList[1][0] != "http://tracker2.example.com/announce" {
		t.Error("Expected announce-list to be 'http://tracker2.example.com/announce', got ", metaInfo.AnnounceList[1][0])
	}
	if metaInfo.Info.Length != 123456 {
		t.Error("Expected length to be 123456, got ", metaInfo.Info.Length)
	}
	if metaInfo.Info.Name != "spam" {
		t.Error("Expected name to be 'spam', got ", metaInfo.Info.Name)
	}
	if metaInfo.Info.PieceLength != 16384 {
		t.Error("Expected piece length to be 16384, got ", metaInfo.Info.PieceLength)
	}
	if string(metaInfo.Info.Pieces[0][:]) != "aaaaaaaaaaaaaaaaaaaa" {
		t.Error("Expected pieces to be 'aaaaaaaaaaaaaaaaaaaa', got ", metaInfo.Info.Pieces)
	}
}

func TestParseMetaInfoFile(t *testing.T) {
	file, err := os.Open("../download/Alpine Standard 3.16.2 x86 64 ISO.torrent")
	if err != nil {
		t.Error("Error opening file: ", err)
	}
	defer file.Close()
	// Read the file into a byte array
	data := make([]byte, 1024*1024)
	n, err := file.Read(data)
	if err != nil && err != io.EOF {
		t.Error("Error reading file: ", err)
	}
	var metaInfo *MetaInfo
	metaInfo, err = ParseMetaInfo(data[:n])
	if err != nil {
		t.Error("Error parsing metainfo: ", err)
	}
	if metaInfo.Announce != "http://linuxtracker.org:2710/00000000000000000000000000000000/announce" {
		t.Error("Expected announce to be 'http://linuxtracker.org:2710/00000000000000000000000000000000/announce', got ", metaInfo.Announce)
	}
	if metaInfo.AnnounceList[0][0] != "http://linuxtracker.org:2710/00000000000000000000000000000000/announce" {
		t.Error("Expected announce-list to be 'http://linuxtracker.org:2710/00000000000000000000000000000000/announce', got ", metaInfo.AnnounceList[0][0])
	}
	if metaInfo.AnnounceList[1][0] != "udp://tracker.opentrackr.org:1337/announce" {
		t.Error("Expected announce-list to be 'udp://tracker.opentrackr.org:1337/announce', got ", metaInfo.AnnounceList[1][0])
	}
	if metaInfo.Info.Length != 156237824 {
		t.Error("Expected length to be 156237824, got ", metaInfo.Info.Length)
	}
	if metaInfo.Info.Name != "alpine-standard-3.16.2-x86_64.iso" {
		t.Error("Expected name to be 'alpine-standard-3.16.2-x86_64.iso', got ", metaInfo.Info.Name)
	}
	if metaInfo.Info.PieceLength != 65536 {
		t.Error("Expected piece length to be 65536, got ", metaInfo.Info.PieceLength)
	}
	piece1 := [20]byte{0x4F, 0x63, 0x2B, 0x38, 0x85, 0xEF, 0x23, 0x02, 0xE2, 0x98, 0xF0, 0xD0, 0x75, 0xAA, 0xA0, 0x3A, 0x17, 0x17, 0x5E, 0xAA}
	// compare byte array
	if !bytes.Equal(metaInfo.Info.Pieces[0][:], piece1[:]) {
		t.Error(piece1, "got ", metaInfo.Info.Pieces[0][:])
	}
}
