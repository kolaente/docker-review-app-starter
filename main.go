package main

import (
	"flag"
	"log"
)

func main() {
	configPath := flag.String("config", "/etc/review-proxy/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Printf("Review proxy starting for domain %s", cfg.Domain)
}
