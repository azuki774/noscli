package config

import (
	"os"
	"strings"
)

// Config represents runtime configuration exposed to the CLI.
type Config struct {
	Timeline TimelineConfig
}

// TimelineConfig holds defaults for the timeline command.
type TimelineConfig struct {
	Relay string
}

// Load reads configuration from environment variables and falls back to defaults.
func Load() Config {
	cfg := Config{
		Timeline: TimelineConfig{
			Relay: "wss://relay-jp.nostr.wirednet.jp",
		},
	}

	if relayEnv := strings.TrimSpace(os.Getenv("NOSCLI_RELAY")); relayEnv != "" {
		cfg.Timeline.Relay = relayEnv
	}

	return cfg
}
