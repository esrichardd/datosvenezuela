package main

import (
	"fmt"
	"os"
	"strconv"
)

// config holds the settings read from environment variables.
type config struct {
	port    int
	dataDir string // directory with the CSV datasets (the repo's data/)
}

// loadConfig reads the configuration from environment variables, applying
// defaults. Invalid values fail fast at startup instead of at first use.
func loadConfig() (config, error) {
	// The default dataDir works for `make run`, which runs from api/.
	cfg := config{port: 8080, dataDir: "../data"}

	if v, ok := os.LookupEnv("PORT"); ok {
		port, err := strconv.Atoi(v)
		if err != nil || port < 1 || port > 65535 {
			return config{}, fmt.Errorf("invalid PORT %q: must be a number between 1 and 65535", v)
		}
		cfg.port = port
	}

	if v, ok := os.LookupEnv("DATA_DIR"); ok {
		cfg.dataDir = v
	}
	if info, err := os.Stat(cfg.dataDir); err != nil || !info.IsDir() {
		return config{}, fmt.Errorf("invalid DATA_DIR %q: must be an existing directory", cfg.dataDir)
	}

	return cfg, nil
}
