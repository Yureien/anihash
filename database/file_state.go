package database

import (
	"encoding/json"

	"gorm.io/gorm"
)

type FileStateEnum uint8

const (
	FILE_PENDING FileStateEnum = iota
	FILE_AVAILABLE
	FILE_ERROR
	FILE_NOT_FOUND
)

type FileState struct {
	gorm.Model

	FileID *uint32 `gorm:"uniqueIndex:idx_db_file_id"`
	Ed2K   string  `gorm:"index"`
	Size   int64   `gorm:"index"`
	State  uint8
	Error  string
}

func (fs FileState) MarshalJSON() ([]byte, error) {
	var stateName string
	switch FileStateEnum(fs.State) {
	case FILE_PENDING:
		stateName = "FILE_PENDING"
	case FILE_AVAILABLE:
		stateName = "FILE_AVAILABLE"
	case FILE_ERROR:
		stateName = "FILE_ERROR"
	default:
		stateName = "UNKNOWN"
	}

	return json.Marshal(struct {
		FileID *uint32 `json:"file_id"`
		State  string  `json:"state"`
		Error  string  `json:"error"`
	}{
		FileID: fs.FileID,
		State:  stateName,
		Error:  fs.Error,
	})
}

func QueryFileStateByFileID(db *gorm.DB, fileID uint32) (FileState, error) {
	var fileState FileState
	if err := db.Where("file_id = ?", fileID).First(&fileState).Error; err != nil {
		return FileState{}, err
	}
	return fileState, nil
}

func QueryFileStateByEd2KSize(db *gorm.DB, ed2k string, size int64) (FileState, error) {
	var fileState FileState
	if err := db.Where("ed2_k = ? AND size = ?", ed2k, size).First(&fileState).Error; err != nil {
		return FileState{}, err
	}
	return fileState, nil
}

func QueryPendingFiles(db *gorm.DB) ([]FileState, error) {
	var fileStates []FileState
	if err := db.Where("state = ?", uint8(FILE_PENDING)).Find(&fileStates).Error; err != nil {
		return nil, err
	}
	return fileStates, nil
}

func CreatePendingFileState(db *gorm.DB, ed2k string, size int64) (FileState, error) {
	fileState := FileState{
		Ed2K:  ed2k,
		Size:  size,
		State: uint8(FILE_PENDING),
	}
	if err := db.Create(&fileState).Error; err != nil {
		return FileState{}, err
	}
	return fileState, nil
}

func UpdateErroredFileState(db *gorm.DB, ed2k string, size int64, state FileStateEnum, error string) error {
	return db.Model(&FileState{}).Where("ed2_k = ? AND size = ?", ed2k, size).Updates(map[string]any{
		"state": uint8(state),
		"error": error,
	}).Error
}

func UpdateAvailableFileState(db *gorm.DB, fileID uint, ed2k string, size int64) error {
	return db.Model(&FileState{}).Where("ed2_k = ? AND size = ?", ed2k, size).Updates(map[string]any{
		"state":   uint8(FILE_AVAILABLE),
		"file_id": fileID,
	}).Error
}
