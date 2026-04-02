package geoip

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testDBPath(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	mmdbPath := filepath.Join(dir, "testdata", "test.mmdb")
	gzPath := filepath.Join(dir, "testdata", "test.mmdb.gz")

	// If mmdb exists, use it
	if _, err := os.Stat(mmdbPath); err == nil {
		return mmdbPath
	}

	// If gz exists, decompress once
	if _, err := os.Stat(gzPath); err == nil {
		t.Log("decompressing test.mmdb.gz...")
		f, err := os.Open(gzPath)
		if err != nil {
			t.Fatalf("open gz: %v", err)
		}
		defer f.Close()
		gr, err := gzip.NewReader(f)
		if err != nil {
			t.Fatalf("gzip reader: %v", err)
		}
		defer gr.Close()
		out, err := os.Create(mmdbPath)
		if err != nil {
			t.Fatalf("create mmdb: %v", err)
		}
		defer out.Close()
		if _, err := io.Copy(out, gr); err != nil {
			t.Fatalf("decompress: %v", err)
		}
		t.Log("test.mmdb ready")
		return mmdbPath
	}

	return mmdbPath // will trigger skip in tests
}

func TestResolver_Lookup(t *testing.T) {
	dbPath := testDBPath(t)
	if _, err := os.Stat(dbPath); err != nil {
		t.Skip("test mmdb not found, skipping")
	}

	r, err := NewResolver(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	loc := r.Lookup("8.8.8.8")
	if loc.Country == "" {
		t.Error("expected country for 8.8.8.8")
	}
	t.Logf("8.8.8.8 -> %s, %s", loc.Country, loc.City)
}

func TestResolver_LookupPrivateIP(t *testing.T) {
	dbPath := testDBPath(t)
	if _, err := os.Stat(dbPath); err != nil {
		t.Skip("test mmdb not found, skipping")
	}

	r, err := NewResolver(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	loc := r.Lookup("192.168.1.1")
	if loc.Country != "" {
		t.Errorf("expected empty country for private IP, got %s", loc.Country)
	}
}

func TestResolver_LookupInvalidIP(t *testing.T) {
	dbPath := testDBPath(t)
	if _, err := os.Stat(dbPath); err != nil {
		t.Skip("test mmdb not found, skipping")
	}

	r, err := NewResolver(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	loc := r.Lookup("not-an-ip")
	if loc.Country != "" {
		t.Error("expected empty for invalid IP")
	}
}

func TestResolver_LookupEmpty(t *testing.T) {
	dbPath := testDBPath(t)
	if _, err := os.Stat(dbPath); err != nil {
		t.Skip("test mmdb not found, skipping")
	}

	r, err := NewResolver(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	loc := r.Lookup("")
	if loc.Country != "" {
		t.Error("expected empty for empty IP")
	}
}

func TestResolver_KnownIPs(t *testing.T) {
	dbPath := testDBPath(t)
	if _, err := os.Stat(dbPath); err != nil {
		t.Skip("test mmdb not found, skipping")
	}

	r, err := NewResolver(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	loc := r.Lookup("1.1.1.1")
	t.Logf("1.1.1.1 -> %s, %s", loc.Country, loc.City)

	loc2 := r.Lookup("81.2.69.142")
	t.Logf("81.2.69.142 -> %s, %s", loc2.Country, loc2.City)
}

func TestNewResolver_BadPath(t *testing.T) {
	_, err := NewResolver("/nonexistent/path.mmdb")
	if err == nil {
		t.Error("expected error for bad path")
	}
}
