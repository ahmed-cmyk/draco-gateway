package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServiceName string `yaml:"service-name"`
	Server      struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Routes []Route `yaml:"routes"`
}

type Route struct {
	Path        string   `yaml:"path"`
	Target      string   `yaml:"target"`
	StripPrefix bool     `yaml:"strip-prefix"`
	Middlewares []string `yaml:"middlewares"`
}

func (c *Config) LoadData(path string) error {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	return yaml.Unmarshal(data, c)
}
