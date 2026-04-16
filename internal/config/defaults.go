package config

import "github.com/goccy/go-yaml"

const DefaultYAML = `version: 1
language: zh

network:
  proxy:
    mode: auto
    manual_url: ""
  timeout_ms: 5000
  tick_interval_sec: 60
  adaptive_tick:
    enabled: true
    fast_interval_sec: 10
    threshold_ms: 1000
    recovery_threshold_ms: 300
    max_fast_duration_sec: 600

probes:
  https:
    - name: Claude API
      url: https://api.anthropic.com
      notify_on_down: true
    - name: Claude Web
      url: https://claude.ai
      notify_on_down: false
    - name: OpenAI API
      url: https://api.openai.com
      notify_on_down: true
    - name: ChatGPT Web
      url: https://chatgpt.com
      notify_on_down: false
    - name: Gemini API
      url: https://generativelanguage.googleapis.com
      notify_on_down: true
    - name: Gemini Web
      url: https://gemini.google.com
      notify_on_down: false

  statuspage:
    refresh_interval_sec: 300
    notify_on_major_outage: true
    sources:
      - name: Anthropic
        url: https://status.anthropic.com/api/v2/status.json
      - name: OpenAI
        url: https://status.openai.com/api/v2/status.json

  ip:
    public_ip_api: https://api.ipify.org?format=json
    geo_api: http://ip-api.com/json/{ip}
    refresh_every_ticks: 5

ui:
  auto_start: false
  minimize_to_tray_on_close: true
  theme: auto

logging:
  level: info
  retention_days: 30
`

func Default() *Config {
	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(DefaultYAML), cfg); err != nil {
		panic("default YAML invalid: " + err.Error())
	}
	return cfg
}
