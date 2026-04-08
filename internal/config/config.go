package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server struct {
		Port int
	}
}

// Load reads a minimal YAML-like config (no external deps needed).
func Load(path string) (*Config, error) {
	cfg := &Config{}
	cfg.Server.Port = 8080

	f, err := os.Open(path)
	if err != nil {
		return cfg, nil
	}
	defer f.Close()

	var section string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if section == "server" && key == "port" {
			if p, err := strconv.Atoi(val); err == nil {
				cfg.Server.Port = p
			}
		}
	}
	return cfg, scanner.Err()
}
