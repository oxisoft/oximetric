package geoip

import (
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const dbipDownloadURL = "https://download.db-ip.com/free/dbip-city-lite-%d-%02d.mmdb.gz"

type Updater struct {
	dbPath string
	logger *slog.Logger
}

func NewUpdater(dbPath string, logger *slog.Logger) *Updater {
	return &Updater{dbPath: dbPath, logger: logger}
}

func (u *Updater) DatabaseExists() bool {
	_, err := os.Stat(u.dbPath)
	return err == nil
}

func (u *Updater) NeedsUpdate() bool {
	if !u.DatabaseExists() {
		return true
	}
	info, err := os.Stat(u.dbPath)
	if err != nil {
		return true
	}
	modTime := info.ModTime()
	now := time.Now()
	return now.Year() > modTime.Year() || (now.Year() == modTime.Year() && now.Month() > modTime.Month())
}

func (u *Updater) Update() error {
	u.logger.Info("downloading GeoIP database from DB-IP")

	now := time.Now()
	url := fmt.Sprintf(dbipDownloadURL, now.Year(), int(now.Month()))

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		prev := now.AddDate(0, -1, 0)
		url = fmt.Sprintf(dbipDownloadURL, prev.Year(), int(prev.Month()))
		resp, err = http.Get(url)
		if err != nil {
			return fmt.Errorf("download fallback: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	dir := filepath.Dir(u.dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, "geoip-*.mmdb.gz")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("save download: %w", err)
	}
	tmpFile.Close()

	f, err := os.Open(tmpPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	out, err := os.Create(u.dbPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, gzr); err != nil {
		return err
	}

	u.logger.Info("GeoIP database updated", "path", u.dbPath)
	return nil
}
