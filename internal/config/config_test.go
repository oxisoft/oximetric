package config

import (
	"os"
	"testing"
)

const validSecret = "this-is-a-jwt-secret-that-is-at-least-32-characters"

func TestLoad_RequiredFields(t *testing.T) {
	os.Clearenv()
	_, err := Load()
	if err == nil {
		t.Fatal("should fail without required env vars")
	}
}

func TestLoad_AllRequired(t *testing.T) {
	os.Clearenv()
	os.Setenv("OXIMETRIC_ADMIN_USERNAME", "admin")
	os.Setenv("OXIMETRIC_ADMIN_PASSWORD", "pass")
	os.Setenv("OXIMETRIC_JWT_SECRET", validSecret)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AdminUsername != "admin" {
		t.Errorf("expected admin, got %s", cfg.AdminUsername)
	}
	if cfg.DBDriver != "sqlite" {
		t.Errorf("expected sqlite default, got %s", cfg.DBDriver)
	}
	if cfg.Port != "6940" {
		t.Errorf("expected 6940 default, got %s", cfg.Port)
	}
}

func TestLoad_InvalidDriver(t *testing.T) {
	os.Clearenv()
	os.Setenv("OXIMETRIC_ADMIN_USERNAME", "admin")
	os.Setenv("OXIMETRIC_ADMIN_PASSWORD", "pass")
	os.Setenv("OXIMETRIC_JWT_SECRET", validSecret)
	os.Setenv("OXIMETRIC_DB_DRIVER", "mysql")

	_, err := Load()
	if err == nil {
		t.Fatal("should fail with invalid driver")
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	os.Clearenv()
	os.Setenv("OXIMETRIC_ADMIN_USERNAME", "admin")
	os.Setenv("OXIMETRIC_ADMIN_PASSWORD", "pass")
	os.Setenv("OXIMETRIC_JWT_SECRET", validSecret)
	os.Setenv("OXIMETRIC_PORT", "abc")

	_, err := Load()
	if err == nil {
		t.Fatal("should fail with invalid port")
	}
}

func TestLoad_ShortJWTSecret(t *testing.T) {
	os.Clearenv()
	os.Setenv("OXIMETRIC_ADMIN_USERNAME", "admin")
	os.Setenv("OXIMETRIC_ADMIN_PASSWORD", "pass")
	os.Setenv("OXIMETRIC_JWT_SECRET", "short")

	_, err := Load()
	if err == nil {
		t.Fatal("should fail with short JWT secret")
	}
}

func TestLoad_CustomValues(t *testing.T) {
	os.Clearenv()
	os.Setenv("OXIMETRIC_ADMIN_USERNAME", "root")
	os.Setenv("OXIMETRIC_ADMIN_PASSWORD", "rootpass")
	os.Setenv("OXIMETRIC_JWT_SECRET", validSecret)
	os.Setenv("OXIMETRIC_DB_DRIVER", "postgres")
	os.Setenv("OXIMETRIC_DB_DSN", "postgres://localhost/test")
	os.Setenv("OXIMETRIC_PORT", "9090")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBDriver != "postgres" {
		t.Errorf("expected postgres, got %s", cfg.DBDriver)
	}
	if cfg.DBDSN != "postgres://localhost/test" {
		t.Errorf("expected custom dsn, got %s", cfg.DBDSN)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected 9090, got %s", cfg.Port)
	}
}

func TestLoad_TrustedProxies(t *testing.T) {
	os.Clearenv()
	os.Setenv("OXIMETRIC_ADMIN_USERNAME", "admin")
	os.Setenv("OXIMETRIC_ADMIN_PASSWORD", "pass")
	os.Setenv("OXIMETRIC_JWT_SECRET", validSecret)
	os.Setenv("OXIMETRIC_TRUSTED_PROXIES", "10.0.0.1, 10.0.0.2")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.TrustedProxies) != 2 {
		t.Errorf("expected 2 proxies, got %d", len(cfg.TrustedProxies))
	}
}
