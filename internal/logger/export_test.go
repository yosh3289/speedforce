package logger

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExportZip_IncludesRecentLogs(t *testing.T) {
	dir := t.TempDir()
	recent := filepath.Join(dir, "probe-"+time.Now().Format("2006-01-02")+".log")
	old := filepath.Join(dir, "probe-2000-01-01.log")
	os.WriteFile(recent, []byte("recent log"), 0600)
	os.WriteFile(old, []byte("old log"), 0600)
	oldTime := time.Now().AddDate(0, 0, -60)
	os.Chtimes(old, oldTime, oldTime)

	lg, err := New(Options{Dir: dir, Level: "info", RetentionDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	outDir := t.TempDir()
	outPath, err := lg.ExportZip(outDir, 7)
	if err != nil {
		t.Fatal(err)
	}

	zr, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	var recentFound, oldFound bool
	for _, f := range zr.File {
		if filepath.Base(f.Name) == filepath.Base(recent) {
			recentFound = true
		}
		if filepath.Base(f.Name) == filepath.Base(old) {
			oldFound = true
		}
	}
	if !recentFound {
		t.Error("recent log not in zip")
	}
	if oldFound {
		t.Error("old log should not be in 7-day export")
	}
}
