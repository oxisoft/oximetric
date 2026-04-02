package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oxisoft/oximetric/internal/auth"
	"github.com/oxisoft/oximetric/internal/config"
	"github.com/oxisoft/oximetric/internal/geoip"
	"github.com/oxisoft/oximetric/internal/model"
	"github.com/oxisoft/oximetric/internal/server"
	"github.com/oxisoft/oximetric/internal/storage"
)

var Version = "dev"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	setLogLevel(logger, cfg.LogLevel)

	// Initialize storage
	var store storage.Store
	switch cfg.DBDriver {
	case "sqlite":
		store, err = storage.NewSQLite(cfg.DBDSN)
	case "postgres":
		store, err = storage.NewPostgres(cfg.DBDSN)
	}
	if err != nil {
		logger.Error("failed to initialize database", "driver", cfg.DBDriver, "error", err)
		os.Exit(1)
	}
	defer store.Close()
	logger.Info("database initialized", "driver", cfg.DBDriver)

	// Bootstrap admin
	if err := bootstrapAdmin(store, cfg, logger); err != nil {
		logger.Error("failed to bootstrap admin", "error", err)
		os.Exit(1)
	}

	// Initialize GeoIP
	updater := geoip.NewUpdater(cfg.GeoIPDBPath, logger)
	if !updater.DatabaseExists() {
		logger.Info("GeoIP database not found, downloading...")
		if err := updater.Update(); err != nil {
			logger.Warn("failed to download GeoIP database, continuing without geo lookup", "error", err)
		}
	}

	var geo *geoip.Resolver
	if updater.DatabaseExists() {
		geo, err = geoip.NewResolver(cfg.GeoIPDBPath)
		if err != nil {
			logger.Warn("failed to open GeoIP database", "error", err)
		} else {
			defer geo.Close()
			logger.Info("GeoIP initialized", "path", cfg.GeoIPDBPath)
		}
	}

	// Auth service
	authSvc := auth.NewService(cfg.JWTSecret)

	// HTTP server
	srv := server.New(store, authSvc, geo, cfg, Version, logger)
	srv.Addr = net.JoinHostPort("", cfg.Port)

	// Background jobs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go retentionCleanupLoop(ctx, store, logger)
	go geoipUpdateLoop(ctx, updater, logger)

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", "addr", srv.Addr, "version", Version)
		if err := srv.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	sig := <-sigCh
	logger.Info("received signal, shutting down", "signal", sig)
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	logger.Info("server stopped")
}

func bootstrapAdmin(store storage.Store, cfg *config.Config, logger *slog.Logger) error {
	ctx := context.Background()
	user, err := store.GetConsoleUserByLogin(ctx, cfg.AdminUsername)
	if err == sql.ErrNoRows {
		hash, err := auth.HashPassword(cfg.AdminPassword)
		if err != nil {
			return err
		}
		user = &model.ConsoleUser{
			Username:     cfg.AdminUsername,
			PasswordHash: hash,
			Role:         "admin",
		}
		if err := store.CreateConsoleUser(ctx, user); err != nil {
			return err
		}
		logger.Info("default admin created", "username", cfg.AdminUsername)
		return nil
	}
	if err != nil {
		return err
	}

	// Update password if changed, reset 2FA
	if !auth.CheckPassword(user.PasswordHash, cfg.AdminPassword) {
		hash, err := auth.HashPassword(cfg.AdminPassword)
		if err != nil {
			return err
		}
		user.PasswordHash = hash
		user.TOTPEnabled = false
		user.TOTPSecret = nil
		if err := store.UpdateConsoleUser(ctx, user); err != nil {
			return err
		}
		logger.Info("default admin password updated and 2FA reset", "username", cfg.AdminUsername)
	}
	return nil
}

func retentionCleanupLoop(ctx context.Context, store storage.Store, logger *slog.Logger) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run once at startup after a short delay
	time.Sleep(10 * time.Second)
	runRetentionCleanup(ctx, store, logger)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runRetentionCleanup(ctx, store, logger)
		}
	}
}

func runRetentionCleanup(ctx context.Context, store storage.Store, logger *slog.Logger) {
	projects, err := store.GetProjectsWithRetention(ctx)
	if err != nil {
		logger.Warn("retention cleanup: failed to get projects", "error", err)
		return
	}
	for _, p := range projects {
		before := time.Now().AddDate(0, 0, -p.RetentionDays)
		deleted, err := store.CleanupEvents(ctx, p.ID, before)
		if err != nil {
			logger.Warn("retention cleanup failed", "project", p.Name, "error", err)
			continue
		}
		if deleted > 0 {
			logger.Info("retention cleanup", "project", p.Name, "deleted", deleted)
		}
	}
}

func geoipUpdateLoop(ctx context.Context, updater *geoip.Updater, logger *slog.Logger) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if updater.NeedsUpdate() {
				if err := updater.Update(); err != nil {
					logger.Warn("GeoIP update failed", "error", err)
				}
			}
		}
	}
}

func setLogLevel(logger *slog.Logger, level string) {
	// Logger is already created with default level; for production
	// this would use a leveler. Keeping simple for now.
	_ = logger
	_ = level
}
