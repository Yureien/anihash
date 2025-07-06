package anidb

type File struct {
	// FMASK DATA
	FileID          uint32
	AnimeID         uint32
	EpisodeID       uint32
	GroupID         uint32
	State           uint16
	Size            int
	Ed2K            string
	MD5             string
	SHA1            string
	CRC             string
	Quality         string
	Source          string
	AudioCodec      string
	AudioBitrate    uint32
	VideoCodec      string
	VideoBitrate    uint32
	VideoResolution string
	Extension       string

	// AMASK DATA
	Year         string
	Type         string
	RomajiName   string
	EnglishName  string
	EpNum        string
	EpName       string
	EpRomajiName string
	GroupName    string
}
