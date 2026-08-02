package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultDigestCheckInterval = 30 * time.Second

type Config struct {
	Domain              string        `yaml:"domain"`
	ComposeTemplate     string        `yaml:"compose_template"`
	TargetService       string        `yaml:"target_service"`
	TargetPort          int           `yaml:"target_port"`
	IdleTimeout         time.Duration `yaml:"idle_timeout"`
	DigestCheckInterval time.Duration `yaml:"digest_check_interval"`
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
	if cfg.DigestCheckInterval <= 0 {
		cfg.DigestCheckInterval = defaultDigestCheckInterval
	}
	return &cfg, nil
}
