package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed locales/*.json
var localesFS embed.FS

type Translator struct {
	mu       sync.RWMutex
	locale   string
	primary  map[string]string
	fallback map[string]string
}

func New(locale string) (*Translator, error) {
	fallback, err := loadLocale("en")
	if err != nil {
		return nil, err
	}
	primary := fallback
	if locale != "en" {
		p, err := loadLocale(locale)
		if err == nil {
			primary = p
		}
	}
	return &Translator{
		locale:   locale,
		primary:  primary,
		fallback: fallback,
	}, nil
}

func loadLocale(locale string) (map[string]string, error) {
	data, err := localesFS.ReadFile(fmt.Sprintf("locales/%s.json", locale))
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// T returns translation for key. Optional params does {{var}} substitution.
func (tr *Translator) T(key string, params ...map[string]string) string {
	tr.mu.RLock()
	val, ok := tr.primary[key]
	if !ok {
		val, ok = tr.fallback[key]
	}
	tr.mu.RUnlock()
	if !ok {
		return key
	}
	if len(params) > 0 {
		for k, v := range params[0] {
			val = strings.ReplaceAll(val, "{{"+k+"}}", v)
		}
	}
	return val
}

func (tr *Translator) SetLocale(locale string) error {
	primary, err := loadLocale(locale)
	if err != nil {
		return err
	}
	tr.mu.Lock()
	tr.locale = locale
	tr.primary = primary
	tr.mu.Unlock()
	return nil
}

func (tr *Translator) Locale() string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.locale
}
