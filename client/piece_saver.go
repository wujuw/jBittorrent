package client

import (
	"log"
	"os"
	"path/filepath"
)

type PieceSaver struct {
	file *os.File
	bitfieldFile *os.File
}

func NewPieceSaver(metaInfo *MetaInfo, downloadDir string, bitfieldDir string) (*PieceSaver, error) {
	filePath := downloadDir + "/" + metaInfo.Info.Name
	bitfieldFilePath :=  bitfieldDir + "/" + metaInfo.Info.Name + ".bitfield"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// create file
		file, err := create(filePath)
		if err != nil {
			log.Fatal("could not create file: ", filePath, " error: ", err)
			return nil, err
		}
		file.Truncate(int64(metaInfo.Info.Length))
		file.Close()

		// delete exist bitfield file
		if _, err := os.Stat(bitfieldFilePath); err == nil {
			err := os.Remove(bitfieldFilePath)
			if err != nil {
				log.Fatal("could not delete bitfield file: ", bitfieldFilePath)
				return nil, err
			}
		}
	}
	file, err := os.OpenFile(filePath, os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal("Error opening file: ", err)
		return nil, err
	}
	if err := os.MkdirAll(bitfieldDir, 0777); err != nil {
			return nil, err
	}
	bifieldFile, err := os.OpenFile(bitfieldFilePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal("Error opening file: ", err)
		return nil, err
	}

	return &PieceSaver{
		file: file,
		bitfieldFile: bifieldFile,
	}, nil
}

func (ps *PieceSaver) SavePiece(saveTask SavePieceTask, bitfield []byte) error {
	_, err := ps.file.WriteAt(saveTask.Piece, int64(saveTask.PieceIndex * saveTask.FixedPieceLength))
	if err != nil {
		log.Fatal("Error writing to file: ", err)
		return err
	}
	_, err = ps.bitfieldFile.WriteAt(bitfield, 0)
	return err
}

func (ps *PieceSaver) Close() {
	ps.file.Close()
	ps.bitfieldFile.Close()
}

func GetBitfield(metaInfo *MetaInfo, downloadDir string, bitfieldDir string) []byte {
	bitfieldFilePath :=  bitfieldDir + "/" + metaInfo.Info.Name + ".bitfield"

	if _, err := os.Stat(downloadDir + "/" + metaInfo.Info.Name); os.IsNotExist(err) {
		if _, err := os.Stat(bitfieldFilePath); err == nil {
			err := os.Remove(bitfieldFilePath)
			if err != nil {
				log.Fatal("could not delete bitfield file: ", bitfieldFilePath)
				return nil
			}
		}
		return make([]byte, 1+len(metaInfo.Info.Pieces)/8)
	}

	
	if stat, err := os.Stat(bitfieldFilePath); os.IsNotExist(err) {
		return make([]byte, 1+len(metaInfo.Info.Pieces)/8)
	} else if stat.Size() == 0 {
		return make([]byte, 1+len(metaInfo.Info.Pieces)/8)
	}
	file, err := os.OpenFile(bitfieldFilePath, os.O_RDONLY, 0666)
	if err != nil {
		log.Fatal("Error opening file: ", err)
		return nil
	}
	defer file.Close()
	bitfield := make([]byte, 1+len(metaInfo.Info.Pieces)/8)
	_, err = file.ReadAt(bitfield, 0)
	if err != nil {
		log.Fatal("Error reading file: ", err)
		return nil
	}
	return bitfield
}

func create(p string) (*os.File, error) {
    if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
        return nil, err
    }
    return os.Create(p)
}