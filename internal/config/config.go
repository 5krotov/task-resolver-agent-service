package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

type Config struct {
	HTTPConfig `yaml:"http" validate:"required"`
}

type HTTPConfig struct {
	Addr string `yaml:"addr" validate:"required"`
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	return nil
}
