package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/oxisoft/oximetric/internal/auth"
	"github.com/oxisoft/oximetric/internal/config"
	"github.com/oxisoft/oximetric/internal/geoip"
	"github.com/oxisoft/oximetric/internal/model"
	"github.com/oxisoft/oximetric/internal/storage"
)

type testEnv struct {
	ts       *httptest.Server
	store    storage.Store
	adminJWT string
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	f, err := os.CreateTemp("", "oximetric-server-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	store, err := storage.NewSQLite(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	hash, _ := auth.HashPassword("admin123")
	ctx := context.Background()
	store.CreateConsoleUser(ctx, &model.ConsoleUser{
		Username: "admin", PasswordHash: hash, Role: "admin",
	})

	// Try to load GeoIP for more realistic tests
	var geo *geoip.Resolver
	_, thisFile, _, _ := runtime.Caller(0)
	geoDBPath := filepath.Join(filepath.Dir(thisFile), "..", "geoip", "testdata", "test.mmdb")
	if _, err := os.Stat(geoDBPath); err == nil {
		geo, _ = geoip.NewResolver(geoDBPath)
		if geo != nil {
			t.Cleanup(func() { geo.Close() })
		}
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authSvc := auth.NewService("test-secret-that-is-at-least-32-chars-long")
	cfg := &config.Config{TrustedProxies: nil}
	srv := New(store, authSvc, geo, cfg, "test-version", logger)
	ts := httptest.NewServer(srv.Handler)
	// Close order matters: httptest server first, then DB
	t.Cleanup(func() { ts.Close(); store.Close() })

	env := &testEnv{ts: ts, store: store}
	env.login(t)
	return env
}

func (e *testEnv) login(t *testing.T) {
	t.Helper()
	resp := e.post(t, "/api/v1/auth/login", map[string]string{
		"login": "admin", "password": "admin123",
	}, "")
	var body model.LoginResponse
	e.decode(t, resp, &body)
	e.adminJWT = body.Token
}

// --- Health ---

func TestServer_Health(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.get(t, "/api/v1/health", "")
	if resp.StatusCode != 200 && resp.StatusCode != 503 {
		t.Fatalf("expected 200 or 503, got %d", resp.StatusCode)
	}
	var body model.HealthResponse
	env.decode(t, resp, &body)
	if body.Database != "ok" {
		t.Errorf("expected database ok, got %s", body.Database)
	}
	if body.Version != "test-version" {
		t.Errorf("expected test-version, got %s", body.Version)
	}
}

// --- Auth ---

func TestServer_Auth_Login(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/auth/login", map[string]string{
		"login": "admin", "password": "admin123",
	}, "")
	env.expectStatus(t, resp, 200)
	var body model.LoginResponse
	env.decode(t, resp, &body)
	if body.Token == "" {
		t.Error("expected token")
	}
	if body.User == nil || body.User.Username != "admin" {
		t.Error("expected admin user")
	}
}

func TestServer_Auth_LoginBadPassword(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/auth/login", map[string]string{
		"login": "admin", "password": "wrong",
	}, "")
	env.expectStatus(t, resp, 401)
}

func TestServer_Auth_LoginBadBody(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/auth/login", "invalid json{{{", "")
	env.expectStatus(t, resp, 400)
}

func TestServer_Auth_LoginMissingFields(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/auth/login", map[string]string{"login": "admin"}, "")
	env.expectStatus(t, resp, 400)
}

func TestServer_Auth_Me(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.get(t, "/api/v1/auth/me", env.adminJWT)
	env.expectStatus(t, resp, 200)
	var user model.ConsoleUser
	env.decode(t, resp, &user)
	if user.Username != "admin" {
		t.Errorf("expected admin, got %s", user.Username)
	}
}

func TestServer_Auth_MeUnauthorized(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.get(t, "/api/v1/auth/me", "")
	env.expectStatus(t, resp, 401)
}

func TestServer_Auth_Logout(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/auth/logout", nil, env.adminJWT)
	env.expectStatus(t, resp, 200)
}

func TestServer_Auth_ChangePassword(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.put(t, "/api/v1/auth/password", map[string]string{
		"current_password": "admin123", "new_password": "newpassword1",
	}, env.adminJWT)
	env.expectStatus(t, resp, 200)

	// Old password should fail
	resp2 := env.post(t, "/api/v1/auth/login", map[string]string{
		"login": "admin", "password": "admin123",
	}, "")
	env.expectStatus(t, resp2, 401)

	// New password should work
	resp3 := env.post(t, "/api/v1/auth/login", map[string]string{
		"login": "admin", "password": "newpassword1",
	}, "")
	env.expectStatus(t, resp3, 200)
}

func TestServer_Auth_ChangePasswordWrongCurrent(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.put(t, "/api/v1/auth/password", map[string]string{
		"current_password": "wrong", "new_password": "newpassword1",
	}, env.adminJWT)
	env.expectStatus(t, resp, 401)
}

func TestServer_Auth_TOTPSetup(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/auth/totp/setup", nil, env.adminJWT)
	env.expectStatus(t, resp, 200)
	var body model.TOTPSetupResponse
	env.decode(t, resp, &body)
	if body.Secret == "" {
		t.Error("expected TOTP secret")
	}
	if body.URI == "" {
		t.Error("expected TOTP URI")
	}
}

func TestServer_Auth_TOTPEnableBadCode(t *testing.T) {
	env := setupTestEnv(t)
	// Setup first
	env.post(t, "/api/v1/auth/totp/setup", nil, env.adminJWT)
	// Enable with bad code
	resp := env.post(t, "/api/v1/auth/totp/enable", map[string]string{"code": "000000", "password": "admin123"}, env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Auth_TOTPEnableNoSetup(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/auth/totp/enable", map[string]string{"code": "123456", "password": "admin123"}, env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Auth_TOTPDisable(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/auth/totp/disable", map[string]string{
		"password": "admin123",
	}, env.adminJWT)
	env.expectStatus(t, resp, 200)
}

func TestServer_Auth_TOTPDisableWrongPassword(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/auth/totp/disable", map[string]string{
		"password": "wrong",
	}, env.adminJWT)
	env.expectStatus(t, resp, 401)
}

// --- Projects ---

func TestServer_Projects_CRUD(t *testing.T) {
	env := setupTestEnv(t)

	// Create
	resp := env.post(t, "/api/v1/projects", map[string]string{"name": "App1"}, env.adminJWT)
	env.expectStatus(t, resp, 201)
	var p model.Project
	env.decode(t, resp, &p)
	if p.Name != "App1" {
		t.Errorf("expected App1, got %s", p.Name)
	}
	id := p.ID

	// List
	resp2 := env.get(t, "/api/v1/projects", env.adminJWT)
	env.expectStatus(t, resp2, 200)
	var projects []model.Project
	env.decode(t, resp2, &projects)
	if len(projects) == 0 {
		t.Error("expected projects")
	}

	// Update
	resp3 := env.put(t, fmt.Sprintf("/api/v1/projects/%d", id),
		map[string]interface{}{"name": "App1-updated", "retention_days": 30}, env.adminJWT)
	env.expectStatus(t, resp3, 200)
	var p2 model.Project
	env.decode(t, resp3, &p2)
	if p2.Name != "App1-updated" {
		t.Errorf("expected App1-updated, got %s", p2.Name)
	}
	if p2.RetentionDays != 30 {
		t.Errorf("expected 30, got %d", p2.RetentionDays)
	}

	// Delete
	resp4 := env.delete(t, fmt.Sprintf("/api/v1/projects/%d", id), env.adminJWT)
	env.expectStatus(t, resp4, 200)
}

func TestServer_Projects_CreateEmpty(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/projects", map[string]string{"name": ""}, env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Projects_UpdateNotFound(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.put(t, "/api/v1/projects/99999", map[string]string{"name": "x"}, env.adminJWT)
	env.expectStatus(t, resp, 404)
}

func TestServer_Projects_BadID(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.put(t, "/api/v1/projects/abc", map[string]string{"name": "x"}, env.adminJWT)
	env.expectStatus(t, resp, 400)
}

// --- Tokens ---

func TestServer_Tokens_CRUD(t *testing.T) {
	env := setupTestEnv(t)

	// Create project
	resp := env.post(t, "/api/v1/projects", map[string]string{"name": "TokenApp"}, env.adminJWT)
	var p model.Project
	env.decode(t, resp, &p)

	// Create token
	resp2 := env.post(t, fmt.Sprintf("/api/v1/projects/%d/tokens", p.ID),
		map[string]string{"label": "v1"}, env.adminJWT)
	env.expectStatus(t, resp2, 201)
	var tk model.ProjectToken
	env.decode(t, resp2, &tk)
	if len(tk.Token) != 64 {
		t.Errorf("expected 64 char token, got %d", len(tk.Token))
	}

	// List
	resp3 := env.get(t, fmt.Sprintf("/api/v1/projects/%d/tokens", p.ID), env.adminJWT)
	env.expectStatus(t, resp3, 200)

	// Disable
	resp4 := env.put(t, fmt.Sprintf("/api/v1/projects/%d/tokens/%d/disable", p.ID, tk.ID), nil, env.adminJWT)
	env.expectStatus(t, resp4, 200)

	// Enable
	resp5 := env.put(t, fmt.Sprintf("/api/v1/projects/%d/tokens/%d/enable", p.ID, tk.ID), nil, env.adminJWT)
	env.expectStatus(t, resp5, 200)

	// Delete
	resp6 := env.delete(t, fmt.Sprintf("/api/v1/projects/%d/tokens/%d", p.ID, tk.ID), env.adminJWT)
	env.expectStatus(t, resp6, 200)
}

func TestServer_Tokens_DisableNotFound(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/projects", map[string]string{"name": "X"}, env.adminJWT)
	var p model.Project
	env.decode(t, resp, &p)

	resp2 := env.put(t, fmt.Sprintf("/api/v1/projects/%d/tokens/99999/disable", p.ID), nil, env.adminJWT)
	env.expectStatus(t, resp2, 404)
}

// --- Tracking ---

func TestServer_Tracking_FullFlow(t *testing.T) {
	env := setupTestEnv(t)

	// Create project + token
	resp := env.post(t, "/api/v1/projects", map[string]string{"name": "TrackApp"}, env.adminJWT)
	var p model.Project
	env.decode(t, resp, &p)

	resp2 := env.post(t, fmt.Sprintf("/api/v1/projects/%d/tokens", p.ID),
		map[string]string{"label": "v1"}, env.adminJWT)
	var tk model.ProjectToken
	env.decode(t, resp2, &tk)

	// Register device
	resp3 := env.postWithToken(t, "/api/v1/device", map[string]string{
		"device_id": "dev-srv-1", "platform": "android",
		"os_version": "14", "app_version": "2.0", "locale": "de_DE",
	}, tk.Token)
	env.expectStatus(t, resp3, 200)

	// Device update (upsert)
	resp3b := env.postWithToken(t, "/api/v1/device", map[string]string{
		"device_id": "dev-srv-1", "platform": "android",
		"os_version": "15", "app_version": "3.0", "locale": "de_DE",
	}, tk.Token)
	env.expectStatus(t, resp3b, 200)

	// Track events
	resp4 := env.postWithToken(t, "/api/v1/track", map[string]interface{}{
		"device_id": "dev-srv-1",
		"events": []map[string]interface{}{
			{
				"id": "e1", "name": "purchase", "timestamp": "2026-04-02T13:30:00+03:00",
				"properties": map[string]interface{}{
					"amount":   map[string]interface{}{"type": "float", "value": 9.99},
					"currency": map[string]interface{}{"type": "string", "value": "EUR"},
					"premium":  map[string]interface{}{"type": "bool", "value": true},
					"count":    map[string]interface{}{"type": "int", "value": 5},
					"at":       map[string]interface{}{"type": "datetime", "value": "2026-04-02T10:00:00Z"},
				},
			},
			{"id": "e2", "name": "page_view", "timestamp": "2026-04-02T13:31:00+03:00"},
			{"id": "e3", "name": "page_view", "timestamp": "2026-04-02T13:32:00+03:00"},
		},
	}, tk.Token)
	env.expectStatus(t, resp4, 202)
	var tr model.TrackResponse
	env.decode(t, resp4, &tr)
	if tr.Accepted != 3 {
		t.Errorf("expected 3 accepted, got %d", tr.Accepted)
	}

	// Dedup
	resp5 := env.postWithToken(t, "/api/v1/track", map[string]interface{}{
		"device_id": "dev-srv-1",
		"events":    []map[string]interface{}{{"id": "e1", "name": "purchase", "timestamp": "2026-04-02T13:30:00+03:00"}},
	}, tk.Token)
	env.expectStatus(t, resp5, 202)
	var tr2 model.TrackResponse
	env.decode(t, resp5, &tr2)
	if tr2.Accepted != 0 {
		t.Errorf("expected 0 accepted (dedup), got %d", tr2.Accepted)
	}

	// Identify
	resp6 := env.postWithToken(t, "/api/v1/identify", map[string]string{
		"device_id": "dev-srv-1", "anonymous_id": "sha256hash",
	}, tk.Token)
	env.expectStatus(t, resp6, 200)
	var ir model.IdentifyResponse
	env.decode(t, resp6, &ir)
	if ir.UserID == 0 {
		t.Error("expected user_id")
	}

	// Track again — should pick up user_id from device fallback
	resp7 := env.postWithToken(t, "/api/v1/track", map[string]interface{}{
		"device_id": "dev-srv-1",
		"events":    []map[string]interface{}{{"id": "e4", "name": "logout", "timestamp": "2026-04-02T14:00:00+03:00"}},
	}, tk.Token)
	env.expectStatus(t, resp7, 202)
}

func TestServer_Tracking_Validation(t *testing.T) {
	env := setupTestEnv(t)

	resp := env.post(t, "/api/v1/projects", map[string]string{"name": "ValApp"}, env.adminJWT)
	var p model.Project
	env.decode(t, resp, &p)
	resp2 := env.post(t, fmt.Sprintf("/api/v1/projects/%d/tokens", p.ID), map[string]string{"label": "v1"}, env.adminJWT)
	var tk model.ProjectToken
	env.decode(t, resp2, &tk)

	// Missing device_id
	r1 := env.postWithToken(t, "/api/v1/track", map[string]interface{}{
		"events": []map[string]interface{}{{"id": "x", "name": "y", "timestamp": "2026-04-02T00:00:00Z"}},
	}, tk.Token)
	env.expectStatus(t, r1, 400)

	// Empty events
	r2 := env.postWithToken(t, "/api/v1/track", map[string]interface{}{
		"device_id": "d1", "events": []interface{}{},
	}, tk.Token)
	env.expectStatus(t, r2, 400)

	// Bad body
	r3 := env.postWithToken(t, "/api/v1/track", "not json", tk.Token)
	env.expectStatus(t, r3, 400)

	// No token
	r4 := env.postWithToken(t, "/api/v1/track", map[string]interface{}{
		"device_id": "d1", "events": []interface{}{},
	}, "")
	env.expectStatus(t, r4, 401)

	// Device missing fields
	r5 := env.postWithToken(t, "/api/v1/device", map[string]string{"device_id": "d1"}, tk.Token)
	env.expectStatus(t, r5, 400)

	// Identify missing fields
	r6 := env.postWithToken(t, "/api/v1/identify", map[string]string{"device_id": "d1"}, tk.Token)
	env.expectStatus(t, r6, 400)

	// Disabled token
	tokens, _ := env.store.ListProjectTokens(context.Background(), p.ID)
	env.put(t, fmt.Sprintf("/api/v1/projects/%d/tokens/%d/disable", p.ID, tokens[0].ID), nil, env.adminJWT)
	r7 := env.postWithToken(t, "/api/v1/device", map[string]string{
		"device_id": "d1", "platform": "ios",
	}, tk.Token)
	env.expectStatus(t, r7, 401)
}

// --- Users ---

func TestServer_Users_CRUD(t *testing.T) {
	env := setupTestEnv(t)

	// Create
	resp := env.post(t, "/api/v1/users", map[string]string{
		"username": "viewer1", "password": "password123", "role": "viewer",
	}, env.adminJWT)
	env.expectStatus(t, resp, 201)
	var u model.ConsoleUser
	env.decode(t, resp, &u)

	// Create with email
	email := "mgr@test.com"
	resp1b := env.post(t, "/api/v1/users", map[string]interface{}{
		"username": "manager1", "password": "password123", "role": "manager", "email": &email,
	}, env.adminJWT)
	env.expectStatus(t, resp1b, 201)

	// List
	resp2 := env.get(t, "/api/v1/users", env.adminJWT)
	env.expectStatus(t, resp2, 200)
	var users []model.ConsoleUser
	env.decode(t, resp2, &users)
	if len(users) < 3 {
		t.Errorf("expected at least 3 users, got %d", len(users))
	}

	// Update
	resp3 := env.put(t, fmt.Sprintf("/api/v1/users/%d", u.ID),
		map[string]string{"role": "manager"}, env.adminJWT)
	env.expectStatus(t, resp3, 200)

	// Delete
	resp4 := env.delete(t, fmt.Sprintf("/api/v1/users/%d", u.ID), env.adminJWT)
	env.expectStatus(t, resp4, 200)
}

func TestServer_Users_CreateBadRole(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/users", map[string]string{
		"username": "x", "password": "x", "role": "superadmin",
	}, env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Users_CreateMissing(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/users", map[string]string{"username": "x"}, env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Users_UpdateNotFound(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.put(t, "/api/v1/users/99999", map[string]string{"role": "viewer"}, env.adminJWT)
	env.expectStatus(t, resp, 404)
}

func TestServer_Users_UpdateBadRole(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/users", map[string]string{
		"username": "tmp", "password": "tmppassword1", "role": "viewer",
	}, env.adminJWT)
	var u model.ConsoleUser
	env.decode(t, resp, &u)

	resp2 := env.put(t, fmt.Sprintf("/api/v1/users/%d", u.ID),
		map[string]string{"role": "invalid"}, env.adminJWT)
	env.expectStatus(t, resp2, 400)
}

// --- Role enforcement ---

func TestServer_RoleEnforcement(t *testing.T) {
	env := setupTestEnv(t)

	// Create viewer
	env.post(t, "/api/v1/users", map[string]string{
		"username": "v1", "password": "viewerpass1", "role": "viewer",
	}, env.adminJWT)

	// Login as viewer
	resp := env.post(t, "/api/v1/auth/login", map[string]string{
		"login": "v1", "password": "viewerpass1",
	}, "")
	var lr model.LoginResponse
	env.decode(t, resp, &lr)
	viewerJWT := lr.Token

	// Viewer can list projects
	r1 := env.get(t, "/api/v1/projects", viewerJWT)
	env.expectStatus(t, r1, 200)

	// Viewer cannot create projects
	r2 := env.post(t, "/api/v1/projects", map[string]string{"name": "x"}, viewerJWT)
	env.expectStatus(t, r2, 403)

	// Viewer cannot list users
	r3 := env.get(t, "/api/v1/users", viewerJWT)
	env.expectStatus(t, r3, 403)

	// Viewer cannot create users
	r4 := env.post(t, "/api/v1/users", map[string]string{
		"username": "hack", "password": "hack", "role": "admin",
	}, viewerJWT)
	env.expectStatus(t, r4, 403)

	// Create manager
	env.post(t, "/api/v1/users", map[string]string{
		"username": "m1", "password": "managerpass1", "role": "manager",
	}, env.adminJWT)
	resp2 := env.post(t, "/api/v1/auth/login", map[string]string{
		"login": "m1", "password": "managerpass1",
	}, "")
	var lr2 model.LoginResponse
	env.decode(t, resp2, &lr2)
	mgrJWT := lr2.Token

	// Manager can create projects
	r5 := env.post(t, "/api/v1/projects", map[string]string{"name": "MgrProject"}, mgrJWT)
	env.expectStatus(t, r5, 201)

	// Manager cannot delete projects
	var mgrProj model.Project
	env.decode(t, r5, &mgrProj)
	r5d := env.delete(t, fmt.Sprintf("/api/v1/projects/%d", mgrProj.ID), mgrJWT)
	env.expectStatus(t, r5d, 403)

	// Manager can list users (view-only)
	r7 := env.get(t, "/api/v1/users", mgrJWT)
	env.expectStatus(t, r7, 200)

	// Manager cannot create users
	r8 := env.post(t, "/api/v1/users", map[string]string{
		"username": "hack", "password": "hackpassword1", "role": "viewer",
	}, mgrJWT)
	env.expectStatus(t, r8, 403)

	// Manager can update projects
	pr := env.post(t, "/api/v1/projects", map[string]string{"name": "MgrTest"}, env.adminJWT)
	var proj model.Project
	env.decode(t, pr, &proj)
	r6 := env.put(t, fmt.Sprintf("/api/v1/projects/%d", proj.ID), map[string]string{"name": "Updated"}, mgrJWT)
	env.expectStatus(t, r6, 200)
}

// --- Analytics ---

func TestServer_Analytics_AllEndpoints(t *testing.T) {
	env := setupTestEnv(t)

	// Setup: project, token, device, events
	resp := env.post(t, "/api/v1/projects", map[string]string{"name": "AnalApp"}, env.adminJWT)
	var p model.Project
	env.decode(t, resp, &p)

	resp2 := env.post(t, fmt.Sprintf("/api/v1/projects/%d/tokens", p.ID), map[string]string{"label": "v1"}, env.adminJWT)
	var tk model.ProjectToken
	env.decode(t, resp2, &tk)

	env.postWithToken(t, "/api/v1/device", map[string]string{
		"device_id": "anal-dev", "platform": "ios", "os_version": "18", "app_version": "1.0", "locale": "en",
	}, tk.Token)

	env.postWithToken(t, "/api/v1/track", map[string]interface{}{
		"device_id": "anal-dev",
		"events": []map[string]interface{}{
			{"id": "a1", "name": "click", "timestamp": "2026-04-01T10:00:00Z",
				"properties": map[string]interface{}{
					"page": map[string]interface{}{"type": "string", "value": "home"},
				}},
			{"id": "a2", "name": "click", "timestamp": "2026-04-01T11:00:00Z"},
			{"id": "a3", "name": "view", "timestamp": "2026-04-01T12:00:00Z"},
		},
	}, tk.Token)

	env.postWithToken(t, "/api/v1/identify", map[string]string{
		"device_id": "anal-dev", "anonymous_id": "hash123",
	}, tk.Token)

	qs := "?from=2026-01-01T00:00:00Z&to=2027-01-01T00:00:00Z"

	// Overview
	r1 := env.get(t, fmt.Sprintf("/api/v1/analytics/%d/overview%s", p.ID, qs), env.adminJWT)
	env.expectStatus(t, r1, 200)
	var ov model.AnalyticsOverview
	env.decode(t, r1, &ov)
	if ov.TotalEvents != 3 {
		t.Errorf("expected 3 events, got %d", ov.TotalEvents)
	}

	// Events
	r2 := env.get(t, fmt.Sprintf("/api/v1/analytics/%d/events%s&interval=day", p.ID, qs), env.adminJWT)
	env.expectStatus(t, r2, 200)

	// Event properties
	r3 := env.get(t, fmt.Sprintf("/api/v1/analytics/%d/events/click/properties%s", p.ID, qs), env.adminJWT)
	env.expectStatus(t, r3, 200)

	// Devices
	r4 := env.get(t, fmt.Sprintf("/api/v1/analytics/%d/devices%s", p.ID, qs), env.adminJWT)
	env.expectStatus(t, r4, 200)

	// Geo
	r5 := env.get(t, fmt.Sprintf("/api/v1/analytics/%d/geo%s", p.ID, qs), env.adminJWT)
	env.expectStatus(t, r5, 200)

	// Users
	r6 := env.get(t, fmt.Sprintf("/api/v1/analytics/%d/users%s", p.ID, qs), env.adminJWT)
	env.expectStatus(t, r6, 200)

	// Retention
	r7 := env.get(t, fmt.Sprintf("/api/v1/analytics/%d/retention%s", p.ID, qs), env.adminJWT)
	env.expectStatus(t, r7, 200)

	// Bad project_id
	r8 := env.get(t, fmt.Sprintf("/api/v1/analytics/abc/overview%s", qs), env.adminJWT)
	env.expectStatus(t, r8, 400)
}

func TestServer_Analytics_QueryParams(t *testing.T) {
	env := setupTestEnv(t)

	resp := env.post(t, "/api/v1/projects", map[string]string{"name": "QP"}, env.adminJWT)
	var p model.Project
	env.decode(t, resp, &p)

	// Default params (no from/to)
	r1 := env.get(t, fmt.Sprintf("/api/v1/analytics/%d/overview", p.ID), env.adminJWT)
	env.expectStatus(t, r1, 200)

	// Custom limit/offset
	r2 := env.get(t, fmt.Sprintf("/api/v1/analytics/%d/events?limit=5&offset=0&interval=hour", p.ID), env.adminJWT)
	env.expectStatus(t, r2, 200)

	// Week interval
	r3 := env.get(t, fmt.Sprintf("/api/v1/analytics/%d/events?interval=week", p.ID), env.adminJWT)
	env.expectStatus(t, r3, 200)

	// Month interval
	r4 := env.get(t, fmt.Sprintf("/api/v1/analytics/%d/events?interval=month", p.ID), env.adminJWT)
	env.expectStatus(t, r4, 200)
}

// --- Additional edge cases for coverage ---

func TestServer_Projects_DeleteBadID(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.delete(t, "/api/v1/projects/abc", env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Projects_CreateBadBody(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/projects", "bad json", env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Projects_UpdateBadBody(t *testing.T) {
	env := setupTestEnv(t)
	r := env.post(t, "/api/v1/projects", map[string]string{"name": "X"}, env.adminJWT)
	var p model.Project
	env.decode(t, r, &p)
	resp := env.put(t, fmt.Sprintf("/api/v1/projects/%d", p.ID), "bad", env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Tokens_CreateBadBody(t *testing.T) {
	env := setupTestEnv(t)
	r := env.post(t, "/api/v1/projects", map[string]string{"name": "X"}, env.adminJWT)
	var p model.Project
	env.decode(t, r, &p)
	resp := env.post(t, fmt.Sprintf("/api/v1/projects/%d/tokens", p.ID), "bad", env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Tokens_BadIDs(t *testing.T) {
	env := setupTestEnv(t)
	// Bad project ID
	resp := env.get(t, "/api/v1/projects/abc/tokens", env.adminJWT)
	env.expectStatus(t, resp, 400)

	// Bad token ID on disable
	r := env.post(t, "/api/v1/projects", map[string]string{"name": "X"}, env.adminJWT)
	var p model.Project
	env.decode(t, r, &p)
	resp2 := env.put(t, fmt.Sprintf("/api/v1/projects/%d/tokens/abc/disable", p.ID), nil, env.adminJWT)
	env.expectStatus(t, resp2, 400)

	// Bad token ID on enable
	resp3 := env.put(t, fmt.Sprintf("/api/v1/projects/%d/tokens/abc/enable", p.ID), nil, env.adminJWT)
	env.expectStatus(t, resp3, 400)

	// Bad token ID on delete
	resp4 := env.delete(t, fmt.Sprintf("/api/v1/projects/%d/tokens/abc", p.ID), env.adminJWT)
	env.expectStatus(t, resp4, 400)
}

func TestServer_Tokens_EnableNotFound(t *testing.T) {
	env := setupTestEnv(t)
	r := env.post(t, "/api/v1/projects", map[string]string{"name": "X"}, env.adminJWT)
	var p model.Project
	env.decode(t, r, &p)
	resp := env.put(t, fmt.Sprintf("/api/v1/projects/%d/tokens/99999/enable", p.ID), nil, env.adminJWT)
	env.expectStatus(t, resp, 404)
}

func TestServer_Users_BadID(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.put(t, "/api/v1/users/abc", map[string]string{"role": "viewer"}, env.adminJWT)
	env.expectStatus(t, resp, 400)

	resp2 := env.delete(t, "/api/v1/users/abc", env.adminJWT)
	env.expectStatus(t, resp2, 400)
}

func TestServer_Users_CreateBadBody(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/users", "bad", env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Users_UpdateBadBody(t *testing.T) {
	env := setupTestEnv(t)
	r := env.post(t, "/api/v1/users", map[string]string{
		"username": "tmp2", "password": "tmppassword1", "role": "viewer",
	}, env.adminJWT)
	var u model.ConsoleUser
	env.decode(t, r, &u)
	resp := env.put(t, fmt.Sprintf("/api/v1/users/%d", u.ID), "bad", env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Auth_ChangePasswordBadBody(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.put(t, "/api/v1/auth/password", "bad", env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Auth_TOTPSetupBadBody(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/auth/totp/enable", "bad", env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Auth_TOTPDisableBadBody(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/auth/totp/disable", "bad", env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Tracking_DeviceBadBody(t *testing.T) {
	env := setupTestEnv(t)
	r := env.post(t, "/api/v1/projects", map[string]string{"name": "X"}, env.adminJWT)
	var p model.Project
	env.decode(t, r, &p)
	r2 := env.post(t, fmt.Sprintf("/api/v1/projects/%d/tokens", p.ID), map[string]string{"label": "v1"}, env.adminJWT)
	var tk model.ProjectToken
	env.decode(t, r2, &tk)

	resp := env.postWithToken(t, "/api/v1/device", "bad", tk.Token)
	env.expectStatus(t, resp, 400)
}

func TestServer_Tracking_IdentifyBadBody(t *testing.T) {
	env := setupTestEnv(t)
	r := env.post(t, "/api/v1/projects", map[string]string{"name": "X"}, env.adminJWT)
	var p model.Project
	env.decode(t, r, &p)
	r2 := env.post(t, fmt.Sprintf("/api/v1/projects/%d/tokens", p.ID), map[string]string{"label": "v1"}, env.adminJWT)
	var tk model.ProjectToken
	env.decode(t, r2, &tk)

	resp := env.postWithToken(t, "/api/v1/identify", "bad", tk.Token)
	env.expectStatus(t, resp, 400)
}

func TestServer_Tracking_TooManyEvents(t *testing.T) {
	env := setupTestEnv(t)
	r := env.post(t, "/api/v1/projects", map[string]string{"name": "X"}, env.adminJWT)
	var p model.Project
	env.decode(t, r, &p)
	r2 := env.post(t, fmt.Sprintf("/api/v1/projects/%d/tokens", p.ID), map[string]string{"label": "v1"}, env.adminJWT)
	var tk model.ProjectToken
	env.decode(t, r2, &tk)

	events := make([]map[string]interface{}, 101)
	for i := range events {
		events[i] = map[string]interface{}{"id": fmt.Sprintf("e%d", i), "name": "x", "timestamp": "2026-04-01T00:00:00Z"}
	}
	resp := env.postWithToken(t, "/api/v1/track", map[string]interface{}{
		"device_id": "d1", "events": events,
	}, tk.Token)
	env.expectStatus(t, resp, 400)
}

func TestServer_Tracking_ClientIPHeaders(t *testing.T) {
	env := setupTestEnv(t)
	r := env.post(t, "/api/v1/projects", map[string]string{"name": "X"}, env.adminJWT)
	var p model.Project
	env.decode(t, r, &p)
	r2 := env.post(t, fmt.Sprintf("/api/v1/projects/%d/tokens", p.ID), map[string]string{"label": "v1"}, env.adminJWT)
	var tk model.ProjectToken
	env.decode(t, r2, &tk)

	env.postWithToken(t, "/api/v1/device", map[string]string{
		"device_id": "ip-dev", "platform": "web",
	}, tk.Token)

	// X-Forwarded-For
	body, _ := json.Marshal(map[string]interface{}{
		"device_id": "ip-dev",
		"events":    []map[string]interface{}{{"id": "xff1", "name": "test", "timestamp": "2026-04-01T00:00:00Z"}},
	})
	req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/track", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Token", tk.Token)
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18")
	resp, _ := http.DefaultClient.Do(req)
	env.expectStatus(t, resp, 202)

	// X-Real-IP
	body2, _ := json.Marshal(map[string]interface{}{
		"device_id": "ip-dev",
		"events":    []map[string]interface{}{{"id": "xri1", "name": "test", "timestamp": "2026-04-01T00:00:00Z"}},
	})
	req2, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/track", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Token", tk.Token)
	req2.Header.Set("X-Real-IP", "198.51.100.25")
	resp2, _ := http.DefaultClient.Do(req2)
	env.expectStatus(t, resp2, 202)
}

func TestServer_Health_WithGeoIP(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.get(t, "/api/v1/health", "")
	var body model.HealthResponse
	env.decode(t, resp, &body)
	// Just verify it returns a valid response regardless of GeoIP state
	if body.Database != "ok" {
		t.Errorf("expected database ok, got %s", body.Database)
	}
}

func TestServer_Auth_LoginByEmail(t *testing.T) {
	env := setupTestEnv(t)

	email := "admin@test.com"
	env.post(t, "/api/v1/users", map[string]interface{}{
		"username": "emailuser", "password": "password123", "role": "viewer", "email": &email,
	}, env.adminJWT)

	resp := env.post(t, "/api/v1/auth/login", map[string]string{
		"login": "admin@test.com", "password": "password123",
	}, "")
	env.expectStatus(t, resp, 200)
}

func TestServer_Auth_LoginNotFound(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/auth/login", map[string]string{
		"login": "nonexistent", "password": "password123",
	}, "")
	env.expectStatus(t, resp, 401)
}

// --- Console page serving ---

func TestServer_Console_LoginPage(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.get(t, "/", "")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !bytes.Contains(body, []byte("OXI Metric")) {
		t.Error("login page should contain OXI Metric")
	}
}

func TestServer_Console_DashboardPage(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.get(t, "/dashboard", "")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !bytes.Contains(body, []byte("dashboard.js")) {
		t.Error("dashboard page should include dashboard.js")
	}
}

func TestServer_Console_UnknownPage(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.get(t, "/nonexistent-page", "")
	// Unknown pages return 404 (only registered console pages are served)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestServer_Console_StaticCSS(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.get(t, "/static/css/app.css", "")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "text/css; charset=utf-8" {
		t.Errorf("expected text/css, got %s", ct)
	}
}

func TestServer_Console_StaticJS(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.get(t, "/static/js/app.js", "")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "application/javascript; charset=utf-8" {
		t.Errorf("expected application/javascript, got %s", ct)
	}
}

func TestServer_Console_AllPages(t *testing.T) {
	env := setupTestEnv(t)
	pages := []string{"/events", "/devices", "/geo", "/users-analytics", "/projects", "/console-users", "/account"}
	for _, p := range pages {
		resp := env.get(t, p, "")
		if resp.StatusCode != 200 {
			t.Errorf("page %s: expected 200, got %d", p, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// --- Security headers ---

func TestServer_SecurityHeaders(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.get(t, "/api/v1/health", "")
	if resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options: DENY")
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options: nosniff")
	}
	if resp.Header.Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Error("missing Referrer-Policy")
	}
	csp := resp.Header.Get("Content-Security-Policy")
	if csp == "" {
		t.Error("missing Content-Security-Policy")
	}
}

// --- TOTP brute force protection ---

func TestServer_Auth_TOTPBruteForceProtection(t *testing.T) {
	env := setupTestEnv(t)

	// Setup TOTP for admin
	env.post(t, "/api/v1/auth/totp/setup", nil, env.adminJWT)

	// We can't actually enable TOTP without a valid code, but we can test
	// the login TOTP rate limit by simulating a user with TOTP enabled.
	// Instead, test the handler's TOTP tracking directly by attempting
	// login with wrong TOTP codes after getting totp_required response.

	// First, enable TOTP manually via store
	user, _ := env.store.GetConsoleUserByLogin(context.Background(), "admin")
	user.TOTPEnabled = true
	env.store.UpdateConsoleUser(context.Background(), user)

	// Login should require TOTP
	resp := env.post(t, "/api/v1/auth/login", map[string]string{
		"login": "admin", "password": "admin123",
	}, "")
	env.expectStatus(t, resp, 200)
	var lr model.LoginResponse
	env.decode(t, resp, &lr)
	if !lr.TOTPRequired {
		t.Fatal("expected totp_required")
	}

	// Send wrong TOTP codes — should get 401 for first 5
	for i := 0; i < 5; i++ {
		resp := env.post(t, "/api/v1/auth/login", map[string]string{
			"login": "admin", "password": "admin123", "totp_code": "000000",
		}, "")
		env.expectStatus(t, resp, 401)
	}

	// 6th attempt should be rate limited (429)
	resp2 := env.post(t, "/api/v1/auth/login", map[string]string{
		"login": "admin", "password": "admin123", "totp_code": "000000",
	}, "")
	env.expectStatus(t, resp2, 429)

	// Disable TOTP to not break other tests
	user.TOTPEnabled = false
	user.TOTPSecret = nil
	env.store.UpdateConsoleUser(context.Background(), user)
}

// --- Password minimum length ---

func TestServer_Users_PasswordTooShort(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.post(t, "/api/v1/users", map[string]string{
		"username": "shortpw", "password": "short", "role": "viewer",
	}, env.adminJWT)
	env.expectStatus(t, resp, 400)
}

func TestServer_Auth_ChangePasswordTooShort(t *testing.T) {
	env := setupTestEnv(t)
	resp := env.put(t, "/api/v1/auth/password", map[string]string{
		"current_password": "admin123", "new_password": "short",
	}, env.adminJWT)
	env.expectStatus(t, resp, 400)
}

// --- Helpers ---

func (e *testEnv) get(t *testing.T, path, jwt string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", e.ts.URL+path, nil)
	if jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func (e *testEnv) post(t *testing.T, path string, body interface{}, jwt string) *http.Response {
	t.Helper()
	return e.doReq(t, "POST", path, body, jwt, "")
}

func (e *testEnv) put(t *testing.T, path string, body interface{}, jwt string) *http.Response {
	t.Helper()
	return e.doReq(t, "PUT", path, body, jwt, "")
}

func (e *testEnv) delete(t *testing.T, path, jwt string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("DELETE", e.ts.URL+path, nil)
	if jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func (e *testEnv) postWithToken(t *testing.T, path string, body interface{}, token string) *http.Response {
	t.Helper()
	return e.doReq(t, "POST", path, body, "", token)
}

func (e *testEnv) doReq(t *testing.T, method, path string, body interface{}, jwt, xtoken string) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			r = bytes.NewBufferString(v)
		default:
			b, _ := json.Marshal(v)
			r = bytes.NewReader(b)
		}
	}
	req, _ := http.NewRequest(method, e.ts.URL+path, r)
	req.Header.Set("Content-Type", "application/json")
	if jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}
	if xtoken != "" {
		req.Header.Set("X-Token", xtoken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func (e *testEnv) expectStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected %d, got %d: %s", want, resp.StatusCode, string(body))
	}
}

func (e *testEnv) decode(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode: %v", err)
	}
}
