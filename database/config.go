package database

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type DatabaseConfig struct {
	SQLite *SQLiteConfig `yaml:"sqlite"`
}
