package logger

import (
	"archive/zip"
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
		_ = l.current.Sync()
	}
}

func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.current != nil {
		_ = l.current.Close()
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

func (l *Logger) ExportZip(outDir string, days int) (string, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}
	outPath := filepath.Join(outDir, fmt.Sprintf("speedforce-logs-%s.zip", time.Now().Format("20060102-150405")))
	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	entries, err := os.ReadDir(l.opts.Dir)
	if err != nil {
		return "", err
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "probe-") {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().Before(cutoff) {
			continue
		}
		src, err := os.Open(filepath.Join(l.opts.Dir, e.Name()))
		if err != nil {
			continue
		}
		w, err := zw.Create(e.Name())
		if err != nil {
			src.Close()
			return "", err
		}
		_, _ = io.Copy(w, src)
		src.Close()
	}
	return outPath, nil
}
