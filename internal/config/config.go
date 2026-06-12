package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port       int    `json:"port"`
	Host       string `json:"host"`
	AgentToken string `json:"-"`
	Kubeconfig string `json:"kubeconfig"`
	MaxTimeout int    `json:"max_timeout"`
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:       9090,
		Host:       "127.0.0.1",
		Kubeconfig: "/etc/kubernetes/admin.conf",
		MaxTimeout: 300,
	}

	if v := os.Getenv("AGENT_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Port)
	}

	if v := os.Getenv("AGENT_HOST"); v != "" {
		cfg.Host = v
	}

	if v := os.Getenv("AGENT_KUBECONFIG"); v != "" {
		cfg.Kubeconfig = v
	}

	token := ""

	if tokenFile := os.Getenv("AGENT_TOKEN_FILE"); tokenFile != "" {
		data, err := os.ReadFile(tokenFile)
		if err == nil {
			token = strings.TrimSpace(string(data))
		}
	}

	if token == "" {
		token = os.Getenv("AGENT_TOKEN")
	}

	if token == "" {
		tokenBytes := make([]byte, 32)
		for i := range tokenBytes {
			tokenBytes[i] = 0
		}
		cfg.AgentToken = hex.EncodeToString(tokenBytes)
	} else {
		cfg.AgentToken = token
	}

	return cfg, nil
}
