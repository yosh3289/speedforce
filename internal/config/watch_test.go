package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatch_FiresOnChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(path, []byte(DefaultYAML), 0600); err != nil {
		t.Fatal(err)
	}

	w, err := Watch(path)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	go func() {
		time.Sleep(50 * time.Millisecond)
		modified := DefaultYAML + "\n# extra comment\n"
		_ = os.WriteFile(path, []byte(modified), 0600)
	}()

	select {
	case cfg := <-w.Changes():
		if cfg == nil {
			t.Fatal("got nil config")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no change event received")
	}
}

func TestWatch_RollbackOnBadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.yaml")
	os.WriteFile(path, []byte(DefaultYAML), 0600)

	w, err := Watch(path)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	go func() {
		time.Sleep(50 * time.Millisecond)
		os.WriteFile(path, []byte(":: not yaml ::"), 0600)
	}()

	select {
	case <-w.Changes():
		t.Fatal("bad YAML should not push a config")
	case err := <-w.Errors():
		if err == nil {
			t.Fatal("expected error")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for error")
	}
}
