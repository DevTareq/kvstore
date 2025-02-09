package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds the application configuration.
type Config struct {
	Host               string `json:"host"`
	Port               int    `json:"port"`
	WALPath            string `json:"wal_path"`
	SSTablePath        string `json:"sstable_path"`
	MemtableMaxEntries int    `json:"memtable_max_entries"`
}

// LoadConfig reads the config file or sets defaults.
func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Config file not found, using default settings")
		return &Config{
			Host:               "0.0.0.0",
			Port:               8080,
			WALPath:            "data/wal.log",
			SSTablePath:        "data/sstable.db",
			MemtableMaxEntries: 1000, // default maximum entries for the Memtable
		}, nil
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	// Set defaults if any fields are missing.
	if config.Host == "" {
		config.Host = "0.0.0.0"
	}

	if config.Port == 0 {
		config.Port = 8080
	}
	if config.WALPath == "" {
		config.WALPath = "data/wal.log"
	}
	if config.SSTablePath == "" {
		config.SSTablePath = "data/sstable.db"
	}
	if config.MemtableMaxEntries == 0 {
		config.MemtableMaxEntries = 1000
	}

	return config, nil
}
