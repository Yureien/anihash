package server

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}
