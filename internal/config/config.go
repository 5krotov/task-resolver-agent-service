package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Kafka        KafkaConfig        `yaml:"kafka"`
	DataProvider DataProviderConfig `yaml:"data-provider"`
	Agent        AgentConfig        `yaml:"agent"`
}

type KafkaConfig struct {
	Addr        string `yaml:"addr"`
	Group       string `yaml:"group"`
	TaskTopic   string `yaml:"task_topic"`
	StatusTopic string `yaml:"status_topic"`
}

type DataProviderConfig struct {
	Addr string `yaml:"addr"`
}

type AgentConfig struct {
	HTTP HTTPConfig `yaml:"http"`
}

type HTTPConfig struct {
	Addr string `yaml:"addr"`
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
