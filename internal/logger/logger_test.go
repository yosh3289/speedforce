package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew_WritesToFile(t *testing.T) {
	dir := t.TempDir()
	lg, err := New(Options{Dir: dir, Level: "info", RetentionDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	lg.Info().Str("key", "value").Msg("hello")
	lg.Sync()

	entries, _ := os.ReadDir(dir)
	if len(entries) == 0 {
		t.Fatal("no log file written")
	}
	data, _ := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if !strings.Contains(string(data), "hello") || !strings.Contains(string(data), "value") {
		t.Errorf("log content unexpected: %s", data)
	}
}

func TestPrune_RemovesOldFiles(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "probe-2000-01-01.log")
	os.WriteFile(old, []byte("x"), 0600)
	oldTime := time.Now().AddDate(0, 0, -60)
	os.Chtimes(old, oldTime, oldTime)

	lg, err := New(Options{Dir: dir, Level: "info", RetentionDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	lg.Prune()

	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("old file should be pruned")
	}
}
