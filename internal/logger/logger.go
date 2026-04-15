package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type Options struct {
	Dir           string
	Level         string
	RetentionDays int
}

type Logger struct {
	zerolog.Logger
	opts Options

	mu      sync.Mutex
	current *os.File
	curDate string
}

func New(opts Options) (*Logger, error) {
	if err := os.MkdirAll(opts.Dir, 0755); err != nil {
		return nil, err
	}
	lg := &Logger{opts: opts}
	if err := lg.rotate(); err != nil {
		return nil, err
	}
	return lg, nil
}

func parseLevel(s string) zerolog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

func (l *Logger) rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	date := time.Now().Format("2006-01-02")
	if date == l.curDate && l.current != nil {
		return nil
	}
	if l.current != nil {
		l.current.Close()
	}
	path := filepath.Join(l.opts.Dir, fmt.Sprintf("probe-%s.log", date))
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	l.current = f
	l.curDate = date
	l.Logger = zerolog.New(io.MultiWriter(f)).Level(parseLevel(l.opts.Level)).With().Timestamp().Logger()
	return nil
}

func (l *Logger) Sync() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.current != nil {
		l.current.Sync()
	}
}

func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.current != nil {
		l.current.Close()
		l.current = nil
	}
}

func (l *Logger) Prune() {
	entries, err := os.ReadDir(l.opts.Dir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -l.opts.RetentionDays)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "probe-") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(l.opts.Dir, e.Name()))
		}
	}
}
