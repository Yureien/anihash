package database

import "gorm.io/gorm"

type AniDBFile struct {
	gorm.Model

	FileID          uint32 `gorm:"uniqueIndex:idx_file_id"`
	AnimeID         uint32 `gorm:"index"`
	EpisodeID       uint32 `gorm:"index"`
	GroupID         uint32 `gorm:"index"`
	State           uint16
	Size            int    `gorm:"index"`
	Ed2K            string `gorm:"uniqueIndex:idx_ed2k"`
	MD5             string `gorm:"index"`
	SHA1            string `gorm:"index"`
	CRC             string `gorm:"index"`
	Quality         string
	Source          string
	AudioCodec      string
	AudioBitrate    uint32
	VideoCodec      string
	VideoBitrate    uint32
	VideoResolution string
	Extension       string

	Year         string
	Type         string
	RomajiName   string
	EnglishName  string
	EpNum        string
	EpName       string
	EpRomajiName string
	GroupName    string
}

func QueryFileByED2KSize(db *gorm.DB, ed2k string, size int) (AniDBFile, error) {
	var file AniDBFile
	if err := db.Where("ed2_k = ? AND size = ?", ed2k, size).First(&file).Error; err != nil {
		return AniDBFile{}, err
	}
	return file, nil
}

func CreateFile(db *gorm.DB, file AniDBFile) (uint, error) {
	if err := db.Create(&file).Error; err != nil {
		return 0, err
	}
	return file.ID, nil
}
