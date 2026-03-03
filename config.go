package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Domain          string        `yaml:"domain"`
	ComposeTemplate string        `yaml:"compose_template"`
	TargetService   string        `yaml:"target_service"`
	TargetPort      int           `yaml:"target_port"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}
