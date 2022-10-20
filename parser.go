package jbittorrent

import (
	"errors"
)

type MetaInfo struct {
	Announce string
	AnnounceList [][]string
	Comment string
	CreatedBy string
	CreationDate int
	Info Info
}

type Info struct {
	Files []File
	Length int
	Name string
	PieceLength int
	Pieces [][20]byte
}

type File struct {
	Length int
	Path []string
}

func (metaInfo *MetaInfo) ParseMetaInfo(data []byte) (error) {
	readLen := 0
	if data[readLen] != 'd' {
		return errors.New("not a bencoding dictionary")
	}
	readLen++

	for {
		if data[readLen] == 'e' {
			readLen++
			break
		}
		key, keyLen, err := readString(data[readLen:])
		if err != nil {
			return err
		}
		readLen += keyLen

		var valueLen int
		switch key {
		case "announce":
			metaInfo.Announce, valueLen, err = readString(data[readLen:])
			if err != nil {
				return err
			}
		case "announce-list":
			valueLen, err = metaInfo.readAnnouceList(data[readLen:])
			if err != nil {
				return err
			}
		case "comment":
			metaInfo.Comment, valueLen, err = readString(data[readLen:])
			if err != nil {
				return err
			}
		case "created by":
			metaInfo.CreatedBy, valueLen, err = readString(data[readLen:])
			if err != nil {
				return err
			}
		case "creation date":
			metaInfo.CreationDate, valueLen, err = readInt(data[readLen:])
			if err != nil {
				return err
			}
		case "info":
			valueLen, err = metaInfo.readInfo(data[readLen:])
			if err != nil {
				return err
			}
		default:
			valueLen, err = readUnknown(data[readLen:])
			if err != nil {
				return err
			}
			//return errors.New("unknown key: " + key)
		}
		readLen += valueLen
	}
	
	// key, readBytes, err := readString(data[readLen:])
	// if err != nil {
	// 	return err
	// }
	// if key == "announce" {
	// 	readLen += readBytes
	// 	announce, readBytes, err := readString(data[readLen:])
	// 	if err != nil {
	// 		return err
	// 	}
	// 	meatInfo.Announce = announce
	// 	readLen += readBytes
	// }
	// key, readBytes, err = readString(data[readLen:])
	// if err != nil {
	// 	return err
	// }
	// if key == "announce-list" {
	// 	readLen += readBytes
	// 	readBytes, err := metaInfo.readAnnouceList(data[readLen:])
	// 	if err != nil {
	// 		return err
	// 	}
	// 	readLen += readBytes
	// }
	// if key == "comment" {

	
	return nil
}

func readUnknown(data []byte) (int, error) {
	switch data[0] {
	case 'i':
		_, readLen, err := readInt(data)
		return readLen, err
	case 'l':
		_, readLen, err := readList(data)
		return readLen, err
	case 'd':
		return readDictionary(data)
	default:
		_, readLen, err := readString(data)
		return readLen, err
	}
}

func readDictionary(data []byte) (int, error) {
	if data[0] != 'd' {
		return 0, errors.New("not a bencoding dictionary")
	}
	readLen := 1

	for {
		if data[readLen] == 'e' {
			readLen++
			break
		}
		_, keyLen, err := readString(data[readLen:])
		if err != nil {
			return 0, err
		}
		readLen += keyLen

		valueLen, err := readUnknown(data[readLen:])
		if err != nil {
			return 0, err
		}
		readLen += valueLen
	}

	return readLen, nil
}

func (metaInfo *MetaInfo) readInfo(data []byte) (int, error) {
	if (data[0] != 'd') {
		return 0, errors.New("invalid info")
	}
	readLen := 1
	for {
		if data[readLen] == 'e' {
			readLen++
			break
		}
		key, keyLen, err := readString(data[readLen:])
		if err != nil {
			return 0, err
		}
		readLen += keyLen

		var valueLen int
		switch key {
		case "files":
			valueLen, err = metaInfo.readFiles(data[readLen:])
			if err != nil {
				return 0, err
			}
		case "length":
			metaInfo.Info.Length, valueLen, err = readInt(data[readLen:])
			if err != nil {
				return 0, err
			}
		case "name":
			metaInfo.Info.Name, valueLen, err = readString(data[readLen:])
			if err != nil {
				return 0, err
			}
		case "piece length":
			metaInfo.Info.PieceLength, valueLen, err = readInt(data[readLen:])
			if err != nil {
				return 0, err
			}
		case "pieces":
			valueLen, err = metaInfo.readPieces(data[readLen:])
			if err != nil {
				return 0, err
			}
		default:
			valueLen, err = readUnknown(data[readLen:])
			if err != nil {
				return 0, err
			}
			//return 0, errors.New("unknown key: " + key)
		}
		readLen += valueLen
	}
	return readLen, nil
}

func (metaInfo *MetaInfo) readFiles(data []byte) (int, error) {
	if (data[0] != 'l') {
		return 0, errors.New("invalid files")
	}
	readLen := 1
	for {
		if data[readLen] == 'e' {
			readLen++
			break
		}
		var file File
		if data[readLen] != 'd' {
			return 0, errors.New("invalid file")
		}
		readLen++
		for {
			if data[readLen] == 'e' {
				readLen++
				break
			}
			key, keyLen, err := readString(data[readLen:])
			if err != nil {
				return 0, err
			}
			readLen += keyLen

			var valueLen int
			switch key {
			case "length":
				file.Length, valueLen, err = readInt(data[readLen:])
				if err != nil {
					return 0, err
				}
			case "path":
				file.Path, valueLen, err = readList(data[readLen:])
				if err != nil {
					return 0, err
				}
			default:
				valueLen, err = readUnknown(data[readLen:])
				if err != nil {
					return 0, err
				}
				//return 0, errors.New("unknown key: " + key)
			}
			readLen += valueLen
		}
		metaInfo.Info.Files = append(metaInfo.Info.Files, file)
	}
	return readLen, nil
}

func (metaInfo *MetaInfo) readPieces(data []byte) (int, error) {
	bytesLen, readLen, err := readLengthPrefix(data)
	if err != nil {
		return 0, err
	}
	if bytesLen % 20 != 0 {
		return 0, errors.New("invalid pieces")
	}
	for i:=0; i<bytesLen; i += 20 {
		var piece [20]byte
		copy(piece[:], data[readLen+i:readLen+i+20])
		metaInfo.Info.Pieces = append(metaInfo.Info.Pieces, piece)
	}
	readLen += bytesLen
	return readLen, nil
}

func (metaInfo *MetaInfo) readAnnouceList(data []byte) (int, error) {
	if data[0] != 'l' {
		return 0, errors.New("invalid announce-list")
	}
	readLen := 1
	for {
		if data[readLen] == 'e' {
			readLen++
			break
		}
		if data[readLen] != 'l' {
			return 0, errors.New("invalid announce-list")
		}
		readLen++
		var announceList []string
		for {
			if data[readLen] == 'e' {
				readLen++
				break
			}
			announce, readBytes, err := readString(data[readLen:])
			if err != nil {
				return 0, err
			}
			announceList = append(announceList, announce)
			readLen += readBytes
		}
		metaInfo.AnnounceList = append(metaInfo.AnnounceList, announceList)
	}
	return readLen, nil
}

func readList(data []byte) ([]string, int, error) {
	if data[0] != 'l' {
		return nil, 0, errors.New("invalid list")
	}
	readLen := 1
	var list []string
	for {
		if data[readLen] == 'e' {
			readLen++
			break
		}
		elem, readBytes, err := readString(data[readLen:])
		if err != nil {
			return nil, 0, err
		}
		list = append(list, elem)
		readLen += readBytes
	}
	return list, readLen, nil
}

func readString(data []byte) (string, int, error) {
	lengthPrefix, readLen, err := readLengthPrefix(data)
	if err != nil {
		return "", 0, err
	}
	return string(data[readLen:readLen+int(lengthPrefix)]), readLen+lengthPrefix, nil
}

func readLengthPrefix(data []byte) (int, int, error) {
	readLen := 0
	lengthPrefix := 0
	for _, b := range data {
		if b == ':' {
			readLen++
			break
		} else if b < '0' || b > '9' {
			return 0, 0, errors.New("invalid length-prefix")
		} else {
			lengthPrefix = lengthPrefix * 10 + int(b - '0')
			readLen++
		}
	}
	return lengthPrefix, readLen, nil
}

func readInt(data []byte) (int, int, error) {
	intVal := 0
	readLen := 1 //跳过'i'
	factor := 1
	if data[readLen] == '-' {
		factor = -1
		readLen++
	}
	for _, b := range data[readLen:] {
		if b == 'e' {
			readLen++
			break
		} else if b < '0' || b > '9' {
			return 0, 0, errors.New("invalid integer")
		} else {
			intVal = intVal * 10 + factor * int(b - '0')
			readLen++
		}
	}
	return intVal, readLen, nil
}
