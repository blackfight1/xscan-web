package config

import (
	"encoding/json"
	"os"
)

type AppConfig struct {
	Port          int    `json:"port"`
	DBPath        string `json:"db_path"`
	XscanPath     string `json:"xscan_path"`
	ToolsDir      string `json:"tools_dir"`
	ResultsDir    string `json:"results_dir"`
	MaxConcurrent int    `json:"max_concurrent"`
	AuthToken     string `json:"auth_token"`
}

func DefaultConfig() *AppConfig {
	return &AppConfig{
		Port:          8080,
		DBPath:        "./data/xscan.db",
		XscanPath:     "./xscan/xscan",
		ToolsDir:      "./tools",
		ResultsDir:    "./results",
		MaxConcurrent: 2,
		AuthToken:     "xscan-web-token-2024",
	}
}

func Load(path string) (*AppConfig, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Save default config
			Save(path, cfg)
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func Save(path string, cfg *AppConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
