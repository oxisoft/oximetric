package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AdminUsername string
	AdminPassword string
	DBDriver      string
	DBDSN         string
	Port          string
	JWTSecret     string
	GeoIPDBPath    string
	LogLevel       string
	TrustedProxies []string
	DomainName     string
}

func Load() (*Config, error) {
	cfg := &Config{
		DBDriver:    "sqlite",
		DBDSN:       "./oximetric.db",
		Port:        "6940",
		GeoIPDBPath: "./data/dbip-city-lite.mmdb",
		LogLevel:    "info",
	}

	if v := os.Getenv("OXIMETRIC_ADMIN_USERNAME"); v != "" {
		cfg.AdminUsername = v
	}
	if v := os.Getenv("OXIMETRIC_ADMIN_PASSWORD"); v != "" {
		cfg.AdminPassword = v
	}
	if v := os.Getenv("OXIMETRIC_DB_DRIVER"); v != "" {
		cfg.DBDriver = strings.ToLower(v)
	}
	if v := os.Getenv("OXIMETRIC_DB_DSN"); v != "" {
		cfg.DBDSN = v
	}
	if v := os.Getenv("OXIMETRIC_PORT"); v != "" {
		if _, err := strconv.Atoi(v); err != nil {
			return nil, fmt.Errorf("invalid OXIMETRIC_PORT: %s", v)
		}
		cfg.Port = v
	}
	if v := os.Getenv("OXIMETRIC_JWT_SECRET"); v != "" {
		cfg.JWTSecret = v
	}
	if v := os.Getenv("OXIMETRIC_GEOIP_DB_PATH"); v != "" {
		cfg.GeoIPDBPath = v
	}
	if v := os.Getenv("OXIMETRIC_LOG_LEVEL"); v != "" {
		cfg.LogLevel = strings.ToLower(v)
	}
	if v := os.Getenv("OXIMETRIC_DOMAIN_NAME"); v != "" {
		cfg.DomainName = v
	}
	if v := os.Getenv("OXIMETRIC_TRUSTED_PROXIES"); v != "" {
		for _, p := range strings.Split(v, ",") {
			if t := strings.TrimSpace(p); t != "" {
				cfg.TrustedProxies = append(cfg.TrustedProxies, t)
			}
		}
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.AdminUsername == "" {
		return fmt.Errorf("OXIMETRIC_ADMIN_USERNAME is required")
	}
	if c.AdminPassword == "" {
		return fmt.Errorf("OXIMETRIC_ADMIN_PASSWORD is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("OXIMETRIC_JWT_SECRET is required")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("OXIMETRIC_JWT_SECRET must be at least 32 characters")
	}
	if c.DBDriver != "sqlite" && c.DBDriver != "postgres" {
		return fmt.Errorf("OXIMETRIC_DB_DRIVER must be 'sqlite' or 'postgres', got '%s'", c.DBDriver)
	}
	return nil
}
