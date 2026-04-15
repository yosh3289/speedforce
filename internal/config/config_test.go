package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_UsesDefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(filepath.Join(dir, "missing.yaml"))
	if err != nil {
		t.Fatalf("load err: %v", err)
	}
	if cfg.Network.TimeoutMs != 5000 {
		t.Errorf("timeout default = %d, want 5000", cfg.Network.TimeoutMs)
	}
	if len(cfg.Probes.HTTPS) != 6 {
		t.Errorf("got %d HTTPS probes, want 6 (default set)", len(cfg.Probes.HTTPS))
	}
}

func TestLoad_ParsesYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.yaml")
	err := os.WriteFile(path, []byte(`
version: 1
language: en
network:
  proxy:
    mode: manual
    manual_url: http://127.0.0.1:7890
  timeout_ms: 3000
  tick_interval_sec: 30
  adaptive_tick:
    enabled: false
    fast_interval_sec: 10
    threshold_ms: 1000
    recovery_threshold_ms: 300
    max_fast_duration_sec: 600
probes:
  https:
    - name: Test
      url: https://example.com
      notify_on_down: true
  statuspage:
    refresh_interval_sec: 300
    notify_on_major_outage: true
    sources: []
  ip:
    public_ip_api: https://api.ipify.org?format=json
    geo_api: https://ipapi.co/{ip}/json/
    refresh_every_ticks: 5
ui:
  auto_start: true
  minimize_to_tray_on_close: true
  theme: dark
logging:
  level: debug
  retention_days: 7
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Language != "en" {
		t.Errorf("language = %q", cfg.Language)
	}
	if cfg.Network.Proxy.Mode != "manual" {
		t.Errorf("mode = %q", cfg.Network.Proxy.Mode)
	}
	if cfg.UI.Theme != "dark" {
		t.Errorf("theme = %q", cfg.UI.Theme)
	}
}

func TestLoad_RollbackOnInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte("::: not yaml :::"), 0600)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
}
