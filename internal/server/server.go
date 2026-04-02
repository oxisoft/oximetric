package server

import (
	"log/slog"
	"net/http"

	"github.com/oxisoft/oximetric/internal/auth"
	"github.com/oxisoft/oximetric/internal/config"
	"github.com/oxisoft/oximetric/internal/geoip"
	"github.com/oxisoft/oximetric/internal/handler"
	"github.com/oxisoft/oximetric/internal/middleware"
	"github.com/oxisoft/oximetric/internal/storage"
)

func New(store storage.Store, authSvc *auth.Service, geo *geoip.Resolver, cfg *config.Config, version string, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()

	health := handler.NewHealthHandler(store, geo, version)
	tracking := handler.NewTrackingHandler(store, geo, cfg.TrustedProxies)
	authH := handler.NewAuthHandler(store, authSvc, logger)
	projects := handler.NewProjectsHandler(store, logger)
	users := handler.NewUsersHandler(store, logger)
	analytics := handler.NewAnalyticsHandler(store)

	// Console UI
	consoleh, err := handler.NewConsoleHandler(cfg.DomainName)
	if err != nil {
		logger.Error("failed to initialize console", "error", err)
	} else {
		mux.HandleFunc("GET /robots.txt", consoleh.RobotsTxt)
		mux.HandleFunc("GET /{$}", consoleh.Login)
		mux.HandleFunc("GET /dashboard", consoleh.Page)
		mux.HandleFunc("GET /events", consoleh.Page)
		mux.HandleFunc("GET /devices", consoleh.Page)
		mux.HandleFunc("GET /geo", consoleh.Page)
		mux.HandleFunc("GET /users-analytics", consoleh.Page)
		mux.HandleFunc("GET /projects", consoleh.Page)
		mux.HandleFunc("GET /console-users", consoleh.Page)
		mux.HandleFunc("GET /account", consoleh.Page)
		mux.HandleFunc("GET /help", consoleh.Page)
		mux.HandleFunc("GET /about", consoleh.Page)
		mux.HandleFunc("GET /static/", consoleh.Static)
	}

	// Health
	mux.HandleFunc("GET /api/v1/health", health.Health)

	// Tracking API — token auth + rate limit
	trackingMux := http.NewServeMux()
	trackingMux.HandleFunc("POST /api/v1/track", tracking.Track)
	trackingMux.HandleFunc("POST /api/v1/device", tracking.Device)
	trackingMux.HandleFunc("POST /api/v1/identify", tracking.Identify)
	mux.Handle("POST /api/v1/track",
		middleware.TokenAuth(store, middleware.RateLimit(1000, trackingMux)))
	mux.Handle("POST /api/v1/device",
		middleware.TokenAuth(store, middleware.RateLimit(1000, trackingMux)))
	mux.Handle("POST /api/v1/identify",
		middleware.TokenAuth(store, middleware.RateLimit(1000, trackingMux)))

	// Auth API — login with rate limit, no JWT
	mux.Handle("POST /api/v1/auth/login",
		middleware.RateLimit(10, http.HandlerFunc(authH.Login)))

	// Auth API — JWT required
	jwtMux := http.NewServeMux()
	jwtMux.HandleFunc("POST /api/v1/auth/logout", authH.Logout)
	jwtMux.HandleFunc("GET /api/v1/auth/me", authH.Me)
	jwtMux.HandleFunc("PUT /api/v1/auth/password", authH.ChangePassword)
	jwtMux.HandleFunc("POST /api/v1/auth/totp/setup", authH.TOTPSetup)
	jwtMux.HandleFunc("POST /api/v1/auth/totp/enable", authH.TOTPEnable)
	jwtMux.HandleFunc("POST /api/v1/auth/totp/disable", authH.TOTPDisable)

	// Projects — viewer can list (for analytics selector), manager can create/update, admin can delete
	jwtMux.HandleFunc("GET /api/v1/projects", projects.List)
	jwtMux.Handle("POST /api/v1/projects",
		middleware.RequireRole("manager", http.HandlerFunc(projects.Create)))
	jwtMux.Handle("PUT /api/v1/projects/{id}",
		middleware.RequireRole("manager", http.HandlerFunc(projects.Update)))
	jwtMux.Handle("DELETE /api/v1/projects/{id}",
		middleware.RequireRole("admin", http.HandlerFunc(projects.Delete)))

	// Tokens — manager can create/list/disable/enable, admin can delete
	jwtMux.Handle("GET /api/v1/projects/{id}/tokens",
		middleware.RequireRole("manager", http.HandlerFunc(projects.ListTokens)))
	jwtMux.Handle("POST /api/v1/projects/{id}/tokens",
		middleware.RequireRole("manager", http.HandlerFunc(projects.CreateToken)))
	jwtMux.Handle("PUT /api/v1/projects/{id}/tokens/{token_id}/disable",
		middleware.RequireRole("manager", http.HandlerFunc(projects.DisableToken)))
	jwtMux.Handle("PUT /api/v1/projects/{id}/tokens/{token_id}/enable",
		middleware.RequireRole("manager", http.HandlerFunc(projects.EnableToken)))
	jwtMux.Handle("DELETE /api/v1/projects/{id}/tokens/{token_id}",
		middleware.RequireRole("admin", http.HandlerFunc(projects.DeleteToken)))

	// Users — manager can view (list), admin can create/update/delete
	jwtMux.Handle("GET /api/v1/users",
		middleware.RequireRole("manager", http.HandlerFunc(users.List)))
	jwtMux.Handle("POST /api/v1/users",
		middleware.RequireRole("admin", http.HandlerFunc(users.Create)))
	jwtMux.Handle("PUT /api/v1/users/{id}",
		middleware.RequireRole("admin", http.HandlerFunc(users.Update)))
	jwtMux.Handle("DELETE /api/v1/users/{id}",
		middleware.RequireRole("admin", http.HandlerFunc(users.Delete)))

	// Analytics — viewer minimum (all authenticated users)
	jwtMux.HandleFunc("GET /api/v1/analytics/{project_id}/overview", analytics.Overview)
	jwtMux.HandleFunc("GET /api/v1/analytics/{project_id}/events", analytics.Events)
	jwtMux.HandleFunc("GET /api/v1/analytics/{project_id}/events/{name}/properties", analytics.EventProperties)
	jwtMux.HandleFunc("GET /api/v1/analytics/{project_id}/devices", analytics.Devices)
	jwtMux.HandleFunc("GET /api/v1/analytics/{project_id}/geo", analytics.Geo)
	jwtMux.HandleFunc("GET /api/v1/analytics/{project_id}/users", analytics.Users)
	jwtMux.HandleFunc("GET /api/v1/analytics/{project_id}/retention", analytics.Retention)

	// Wrap all JWT routes
	jwtWrapped := middleware.JWTAuth(authSvc, jwtMux)
	mux.Handle("/api/v1/auth/", jwtWrapped)
	mux.Handle("/api/v1/projects", jwtWrapped)
	mux.Handle("/api/v1/projects/", jwtWrapped)
	mux.Handle("/api/v1/users", jwtWrapped)
	mux.Handle("/api/v1/users/", jwtWrapped)
	mux.Handle("/api/v1/analytics/", jwtWrapped)

	return &http.Server{
		Handler: middleware.SecurityHeaders(middleware.Logging(logger, middleware.BodyLimit(1<<20, mux))),
	}
}
