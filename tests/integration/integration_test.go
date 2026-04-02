package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/oxisoft/oximetric/internal/auth"
	"github.com/oxisoft/oximetric/internal/config"
	"github.com/oxisoft/oximetric/internal/model"
	"github.com/oxisoft/oximetric/internal/server"
	"github.com/oxisoft/oximetric/internal/storage"
)

var dbDriver = flag.String("db", "sqlite", "database driver: sqlite or postgres")
var pgDSN = flag.String("pg-dsn", "", "postgres connection string")

var (
	baseURL  string
	adminJWT string
)

func TestMain(m *testing.M) {
	flag.Parse()

	store, cleanup := setupStore()

	// Bootstrap admin
	ctx := context.Background()
	hash, _ := auth.HashPassword("adminpass")
	store.CreateConsoleUser(ctx, &model.ConsoleUser{
		Username: "admin", PasswordHash: hash, Role: "admin",
	})

	authSvc := auth.NewService("integration-test-secret-at-least-32chars")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := &config.Config{}
	srv := server.New(store, authSvc, nil, cfg, "test", logger)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
	baseURL = "http://" + listener.Addr().String()
	srv.Addr = listener.Addr().String()

	go srv.Serve(listener)

	// Wait for server
	for i := 0; i < 50; i++ {
		resp, err := http.Get(baseURL + "/api/v1/health")
		if err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	code := m.Run()
	srv.Close()
	cleanup()
	os.Exit(code)
}

func setupStore() (storage.Store, func()) {
	switch *dbDriver {
	case "postgres":
		dsn := *pgDSN
		if dsn == "" {
			dsn = os.Getenv("OXIMETRIC_TEST_PG_DSN")
		}
		if dsn == "" {
			dsn = "postgres://oximetric:oximetric@localhost:5432/oximetric_test?sslmode=disable"
		}
		store, err := storage.NewPostgres(dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "postgres: %v\n", err)
			os.Exit(1)
		}
		return store, func() { store.Close() }
	default:
		f, _ := os.CreateTemp("", "oximetric-integration-*.db")
		f.Close()
		store, err := storage.NewSQLite(f.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "sqlite: %v\n", err)
			os.Exit(1)
		}
		return store, func() { store.Close(); os.Remove(f.Name()) }
	}
}

// --- Health ---

func TestHealth(t *testing.T) {
	resp := doGet(t, "/api/v1/health", "")
	// May be 200 or 503 depending on GeoIP availability
	if resp.StatusCode != 200 && resp.StatusCode != 503 {
		t.Fatalf("expected 200 or 503, got %d", resp.StatusCode)
	}

	var body model.HealthResponse
	decodeBody(t, resp, &body)
	if body.Database != "ok" {
		t.Errorf("expected database ok, got %s", body.Database)
	}
	if body.Version != "test" {
		t.Errorf("expected test version, got %s", body.Version)
	}
}

// --- Auth Flow ---

func TestAuth_Login(t *testing.T) {
	resp := doPost(t, "/api/v1/auth/login", map[string]string{
		"login": "admin", "password": "adminpass",
	}, "")
	expectStatus(t, resp, 200)

	var body model.LoginResponse
	decodeBody(t, resp, &body)
	if body.Token == "" {
		t.Fatal("expected JWT token")
	}
	if body.User == nil || body.User.Username != "admin" {
		t.Error("expected admin user in response")
	}
	adminJWT = body.Token
}

func TestAuth_LoginWrongPassword(t *testing.T) {
	resp := doPost(t, "/api/v1/auth/login", map[string]string{
		"login": "admin", "password": "wrong",
	}, "")
	expectStatus(t, resp, 401)
}

func TestAuth_Me(t *testing.T) {
	ensureAdminJWT(t)
	resp := doGet(t, "/api/v1/auth/me", adminJWT)
	expectStatus(t, resp, 200)

	var user model.ConsoleUser
	decodeBody(t, resp, &user)
	if user.Username != "admin" {
		t.Errorf("expected admin, got %s", user.Username)
	}
}

func TestAuth_ChangePassword(t *testing.T) {
	ensureAdminJWT(t)
	resp := doPut(t, "/api/v1/auth/password", map[string]string{
		"current_password": "adminpass", "new_password": "newpassword1",
	}, adminJWT)
	expectStatus(t, resp, 200)

	// Login with new password
	resp2 := doPost(t, "/api/v1/auth/login", map[string]string{
		"login": "admin", "password": "newpassword1",
	}, "")
	expectStatus(t, resp2, 200)

	// Restore original password
	var login model.LoginResponse
	decodeBody(t, resp2, &login)
	doPut(t, "/api/v1/auth/password", map[string]string{
		"current_password": "newpassword1", "new_password": "adminpass",
	}, login.Token)
	adminJWT = ""
}

func TestAuth_Unauthorized(t *testing.T) {
	resp := doGet(t, "/api/v1/auth/me", "")
	expectStatus(t, resp, 401)
}

// --- Project CRUD ---

var testProjectID int

func TestProjects_Create(t *testing.T) {
	ensureAdminJWT(t)
	resp := doPost(t, "/api/v1/projects", map[string]string{"name": "Test App"}, adminJWT)
	expectStatus(t, resp, 201)

	var p model.Project
	decodeBody(t, resp, &p)
	if p.Name != "Test App" {
		t.Errorf("expected Test App, got %s", p.Name)
	}
	testProjectID = p.ID
}

func TestProjects_List(t *testing.T) {
	ensureAdminJWT(t)
	resp := doGet(t, "/api/v1/projects", adminJWT)
	expectStatus(t, resp, 200)

	var projects []model.Project
	decodeBody(t, resp, &projects)
	if len(projects) == 0 {
		t.Error("expected at least 1 project")
	}
}

func TestProjects_Update(t *testing.T) {
	ensureAdminJWT(t)
	resp := doPut(t, fmt.Sprintf("/api/v1/projects/%d", testProjectID),
		map[string]interface{}{"name": "Updated App", "retention_days": 60}, adminJWT)
	expectStatus(t, resp, 200)

	var p model.Project
	decodeBody(t, resp, &p)
	if p.Name != "Updated App" {
		t.Errorf("expected Updated App, got %s", p.Name)
	}
}

// --- Token CRUD ---

var testToken string

func TestTokens_Create(t *testing.T) {
	ensureAdminJWT(t)
	resp := doPost(t, fmt.Sprintf("/api/v1/projects/%d/tokens", testProjectID),
		map[string]string{"label": "v1.0"}, adminJWT)
	expectStatus(t, resp, 201)

	var tk model.ProjectToken
	decodeBody(t, resp, &tk)
	if tk.Token == "" {
		t.Fatal("expected token string")
	}
	if len(tk.Token) != 64 {
		t.Errorf("expected 64 char token, got %d", len(tk.Token))
	}
	testToken = tk.Token
}

func TestTokens_List(t *testing.T) {
	ensureAdminJWT(t)
	resp := doGet(t, fmt.Sprintf("/api/v1/projects/%d/tokens", testProjectID), adminJWT)
	expectStatus(t, resp, 200)
}

func TestTokens_Disable(t *testing.T) {
	ensureAdminJWT(t)
	// Need token ID; list first
	resp := doGet(t, fmt.Sprintf("/api/v1/projects/%d/tokens", testProjectID), adminJWT)
	var body struct {
		Tokens []model.ProjectToken `json:"tokens"`
	}
	decodeBody(t, resp, &body)
	if len(body.Tokens) == 0 {
		t.Fatal("no tokens")
	}
	tokenID := body.Tokens[0].ID

	resp2 := doPut(t, fmt.Sprintf("/api/v1/projects/%d/tokens/%d/disable", testProjectID, tokenID), nil, adminJWT)
	expectStatus(t, resp2, 200)

	// Tracking with disabled token should fail
	resp3 := doPostWithToken(t, "/api/v1/track", map[string]interface{}{
		"device_id": "dev-1", "events": []interface{}{},
	}, testToken)
	expectStatus(t, resp3, 401)

	// Re-enable
	doPut(t, fmt.Sprintf("/api/v1/projects/%d/tokens/%d/enable", testProjectID, tokenID), nil, adminJWT)
}

// --- Tracking API ---

func TestTracking_Device(t *testing.T) {
	resp := doPostWithToken(t, "/api/v1/device", map[string]string{
		"device_id": "test-device-1", "platform": "ios",
		"os_version": "18.0", "app_version": "1.0", "locale": "en_US",
	}, testToken)
	expectStatus(t, resp, 200)
}

func TestTracking_Track(t *testing.T) {
	resp := doPostWithToken(t, "/api/v1/track", map[string]interface{}{
		"device_id": "test-device-1",
		"events": []map[string]interface{}{
			{
				"id": "evt-int-1", "name": "purchase",
				"timestamp": "2026-04-02T13:30:00+03:00",
				"properties": map[string]interface{}{
					"amount":   map[string]interface{}{"type": "float", "value": 29.99},
					"currency": map[string]interface{}{"type": "string", "value": "USD"},
				},
			},
			{
				"id": "evt-int-2", "name": "page_view",
				"timestamp": "2026-04-02T13:31:00+03:00",
			},
		},
	}, testToken)
	expectStatus(t, resp, 202)

	var body model.TrackResponse
	decodeBody(t, resp, &body)
	if body.Accepted != 2 {
		t.Errorf("expected 2 accepted, got %d", body.Accepted)
	}
}

func TestTracking_Dedup(t *testing.T) {
	resp := doPostWithToken(t, "/api/v1/track", map[string]interface{}{
		"device_id": "test-device-1",
		"events": []map[string]interface{}{
			{"id": "evt-int-1", "name": "purchase", "timestamp": "2026-04-02T13:30:00+03:00"},
		},
	}, testToken)
	expectStatus(t, resp, 202)

	var body model.TrackResponse
	decodeBody(t, resp, &body)
	if body.Accepted != 0 {
		t.Errorf("expected 0 accepted (dedup), got %d", body.Accepted)
	}
}

func TestTracking_Identify(t *testing.T) {
	resp := doPostWithToken(t, "/api/v1/identify", map[string]string{
		"device_id": "test-device-1", "anonymous_id": "sha256-user-hash",
	}, testToken)
	expectStatus(t, resp, 200)

	var body model.IdentifyResponse
	decodeBody(t, resp, &body)
	if body.UserID == 0 {
		t.Error("expected user_id")
	}
}

func TestTracking_NoToken(t *testing.T) {
	resp := doPostWithToken(t, "/api/v1/track", map[string]interface{}{
		"device_id": "dev-1", "events": []interface{}{},
	}, "")
	expectStatus(t, resp, 401)
}

// --- User Management ---

func TestUsers_Create(t *testing.T) {
	ensureAdminJWT(t)
	resp := doPost(t, "/api/v1/users", map[string]string{
		"username": "viewer1", "password": "viewerpass1", "role": "viewer",
	}, adminJWT)
	expectStatus(t, resp, 201)
}

func TestUsers_List(t *testing.T) {
	ensureAdminJWT(t)
	resp := doGet(t, "/api/v1/users", adminJWT)
	expectStatus(t, resp, 200)

	var users []model.ConsoleUser
	decodeBody(t, resp, &users)
	if len(users) < 2 {
		t.Errorf("expected at least 2 users, got %d", len(users))
	}
}

func TestUsers_ViewerCannotManage(t *testing.T) {
	// Login as viewer
	resp := doPost(t, "/api/v1/auth/login", map[string]string{
		"login": "viewer1", "password": "viewerpass1",
	}, "")
	expectStatus(t, resp, 200)

	var login model.LoginResponse
	decodeBody(t, resp, &login)

	// Viewer should not be able to create users
	resp2 := doPost(t, "/api/v1/users", map[string]string{
		"username": "hack", "password": "hack", "role": "admin",
	}, login.Token)
	expectStatus(t, resp2, 403)

	// Viewer should not be able to create projects
	resp3 := doPost(t, "/api/v1/projects", map[string]string{"name": "hack"}, login.Token)
	expectStatus(t, resp3, 403)
}

// --- Analytics ---

func TestAnalytics_Overview(t *testing.T) {
	ensureAdminJWT(t)
	resp := doGet(t, fmt.Sprintf("/api/v1/analytics/%d/overview?from=2026-01-01T00:00:00Z&to=2027-01-01T00:00:00Z", testProjectID), adminJWT)
	expectStatus(t, resp, 200)

	var overview model.AnalyticsOverview
	decodeBody(t, resp, &overview)
	if overview.TotalEvents < 2 {
		t.Errorf("expected at least 2 events, got %d", overview.TotalEvents)
	}
}

func TestAnalytics_Events(t *testing.T) {
	ensureAdminJWT(t)
	resp := doGet(t, fmt.Sprintf("/api/v1/analytics/%d/events?from=2026-01-01T00:00:00Z&to=2027-01-01T00:00:00Z", testProjectID), adminJWT)
	expectStatus(t, resp, 200)
}

func TestAnalytics_Devices(t *testing.T) {
	ensureAdminJWT(t)
	resp := doGet(t, fmt.Sprintf("/api/v1/analytics/%d/devices?from=2026-01-01T00:00:00Z&to=2027-01-01T00:00:00Z", testProjectID), adminJWT)
	expectStatus(t, resp, 200)
}

func TestAnalytics_Geo(t *testing.T) {
	ensureAdminJWT(t)
	resp := doGet(t, fmt.Sprintf("/api/v1/analytics/%d/geo?from=2026-01-01T00:00:00Z&to=2027-01-01T00:00:00Z", testProjectID), adminJWT)
	expectStatus(t, resp, 200)
}

func TestAnalytics_Users(t *testing.T) {
	ensureAdminJWT(t)
	resp := doGet(t, fmt.Sprintf("/api/v1/analytics/%d/users?from=2026-01-01T00:00:00Z&to=2027-01-01T00:00:00Z", testProjectID), adminJWT)
	expectStatus(t, resp, 200)
}

// --- Cleanup: Delete project ---

func TestProjects_Delete(t *testing.T) {
	ensureAdminJWT(t)
	resp := doDelete(t, fmt.Sprintf("/api/v1/projects/%d", testProjectID), adminJWT)
	expectStatus(t, resp, 200)
}

// --- Helpers ---

func ensureAdminJWT(t *testing.T) {
	t.Helper()
	if adminJWT != "" {
		return
	}
	resp := doPost(t, "/api/v1/auth/login", map[string]string{
		"login": "admin", "password": "adminpass",
	}, "")
	var body model.LoginResponse
	decodeBody(t, resp, &body)
	adminJWT = body.Token
}

func doGet(t *testing.T, path, jwt string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", baseURL+path, nil)
	if jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func doPost(t *testing.T, path string, body interface{}, jwt string) *http.Response {
	t.Helper()
	return doRequest(t, "POST", path, body, jwt, "")
}

func doPut(t *testing.T, path string, body interface{}, jwt string) *http.Response {
	t.Helper()
	return doRequest(t, "PUT", path, body, jwt, "")
}

func doDelete(t *testing.T, path, jwt string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("DELETE", baseURL+path, nil)
	if jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func doPostWithToken(t *testing.T, path string, body interface{}, token string) *http.Response {
	t.Helper()
	return doRequest(t, "POST", path, body, "", token)
}

func doRequest(t *testing.T, method, path string, body interface{}, jwt, xtoken string) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, baseURL+path, r)
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

func expectStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d: %s", want, resp.StatusCode, string(body))
	}
}

func decodeBody(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}
