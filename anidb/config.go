package anidb

type AniDBConfig struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Address  string `yaml:"address" default:"api.anidb.net:9000"`
}
