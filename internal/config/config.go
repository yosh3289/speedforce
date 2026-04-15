package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Version  int     `yaml:"version"`
	Language string  `yaml:"language"`
	Network  Network `yaml:"network"`
	Probes   Probes  `yaml:"probes"`
	UI       UI      `yaml:"ui"`
	Logging  Logging `yaml:"logging"`
}

type Network struct {
	Proxy        Proxy        `yaml:"proxy"`
	TimeoutMs    int          `yaml:"timeout_ms"`
	TickInterval int          `yaml:"tick_interval_sec"`
	AdaptiveTick AdaptiveTick `yaml:"adaptive_tick"`
}

type Proxy struct {
	Mode      string `yaml:"mode"`
	ManualURL string `yaml:"manual_url"`
}

type AdaptiveTick struct {
	Enabled             bool `yaml:"enabled"`
	FastIntervalSec     int  `yaml:"fast_interval_sec"`
	ThresholdMs         int  `yaml:"threshold_ms"`
	RecoveryThresholdMs int  `yaml:"recovery_threshold_ms"`
	MaxFastDurationSec  int  `yaml:"max_fast_duration_sec"`
}

type Probes struct {
	HTTPS      []HTTPSProbe    `yaml:"https"`
	Statuspage StatuspageBlock `yaml:"statuspage"`
	IP         IPBlock         `yaml:"ip"`
}

type HTTPSProbe struct {
	Name         string `yaml:"name"`
	URL          string `yaml:"url"`
	NotifyOnDown bool   `yaml:"notify_on_down"`
}

type StatuspageBlock struct {
	RefreshIntervalSec  int                `yaml:"refresh_interval_sec"`
	NotifyOnMajorOutage bool               `yaml:"notify_on_major_outage"`
	Sources             []StatuspageSource `yaml:"sources"`
}

type StatuspageSource struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type IPBlock struct {
	PublicIPAPI       string `yaml:"public_ip_api"`
	GeoAPI            string `yaml:"geo_api"`
	RefreshEveryTicks int    `yaml:"refresh_every_ticks"`
}

type UI struct {
	AutoStart             bool   `yaml:"auto_start"`
	MinimizeToTrayOnClose bool   `yaml:"minimize_to_tray_on_close"`
	Theme                 string `yaml:"theme"`
}

type Logging struct {
	Level         string `yaml:"level"`
	RetentionDays int    `yaml:"retention_days"`
}

func Load(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if writeErr := os.WriteFile(path, []byte(DefaultYAML), 0600); writeErr != nil {
				return cfg, fmt.Errorf("write default: %w", writeErr)
			}
			return cfg, nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return cfg, nil
}

func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
