package anidb

type AniDBConfig struct {
	User          string `yaml:"user"`
	Password      string `yaml:"password"`
	ClientName    string `yaml:"client_name" default:"goaniudp"`
	ClientVersion int32  `yaml:"client_version" default:"1"`
	Address       string `yaml:"address" default:"api.anidb.net:9000"`
}
