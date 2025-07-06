package main

import (
	"os"

	"github.com/yureien/anihash/anidb"
	"github.com/yureien/anihash/database"
	"github.com/yureien/anihash/server"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Anidb    anidb.AniDBConfig       `yaml:"anidb"`
	Server   server.ServerConfig     `yaml:"server"`
	Database database.DatabaseConfig `yaml:"database"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
