package config

import (
	"os"

	"github.com/charmbracelet/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ServiceName string `yaml:"service-name"`
	Server      struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Routes []Route `yaml:"routes"`
}

type HeaderConfig struct {
	Set    map[string]string `yaml:"set"`
	Remove []string          `yaml:"remove"`
}

type Route struct {
	Path        string       `yaml:"path"`
	Targets     []string     `yaml:"targets"`
	StripPrefix bool         `yaml:"strip-prefix"`
	Methods     []string     `yaml:"methods"`
	Headers     HeaderConfig `yaml:"headers"`
	Balancer    string       `yaml:"balancer"`
	Middlewares []string     `yaml:"middlewares"`
}

func (c *Config) LoadData(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Errorf("Error loading config: %v", err)
	}

	return yaml.Unmarshal(data, c)
}
