package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
}

func (c Config) CheckConfig(path string) ([]byte, error) {
	return os.ReadFile("config.yaml")
}

func (c *Config) LoadData(data []byte) error {
	return yaml.Unmarshal(data, c)
}
