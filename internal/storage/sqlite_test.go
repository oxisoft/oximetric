package storage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/oxisoft/oximetric/internal/model"
)

func newTestSQLite(t *testing.T) *SQLiteStore {
	t.Helper()
	f, err := os.CreateTemp("", "oximetric-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	store, err := NewSQLite(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestSQLite_Ping(t *testing.T) {
	store := newTestSQLite(t)
	if err := store.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSQLite_ConsoleUsers(t *testing.T) {
	store := newTestSQLite(t)
	ctx := context.Background()

	user := &model.ConsoleUser{
		Username:     "testuser",
		PasswordHash: "hash123",
		Role:         "admin",
	}
	if err := store.CreateConsoleUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	if user.ID == 0 {
		t.Error("user ID should be set")
	}

	got, err := store.GetConsoleUserByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "testuser" {
		t.Errorf("expected testuser, got %s", got.Username)
	}

	got2, err := store.GetConsoleUserByLogin(ctx, "testuser")
	if err != nil {
		t.Fatal(err)
	}
	if got2.ID != user.ID {
		t.Error("login lookup should return same user")
	}

	email := "test@example.com"
	user.Email = &email
	if err := store.UpdateConsoleUser(ctx, user); err != nil {
		t.Fatal(err)
	}

	got3, err := store.GetConsoleUserByLogin(ctx, "test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got3.ID != user.ID {
		t.Error("email login lookup should return same user")
	}

	users, err := store.ListConsoleUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}

	if err := store.DeleteConsoleUser(ctx, user.ID); err != nil {
		t.Fatal(err)
	}

	users, _ = store.ListConsoleUsers(ctx)
	if len(users) != 0 {
		t.Error("user should be deleted")
	}
}

func TestSQLite_Projects(t *testing.T) {
	store := newTestSQLite(t)
	ctx := context.Background()

	p := &model.Project{Name: "Test Project", RetentionDays: 30}
	if err := store.CreateProject(ctx, p); err != nil {
		t.Fatal(err)
	}
	if p.ID == 0 {
		t.Error("project ID should be set")
	}

	got, err := store.GetProject(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Test Project" {
		t.Errorf("expected Test Project, got %s", got.Name)
	}
	if got.RetentionDays != 30 {
		t.Errorf("expected 30 retention days, got %d", got.RetentionDays)
	}

	p.Name = "Updated"
	if err := store.UpdateProject(ctx, p); err != nil {
		t.Fatal(err)
	}

	projects, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].Name != "Updated" {
		t.Error("project should be updated")
	}

	if err := store.DeleteProject(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	projects, _ = store.ListProjects(ctx)
	if len(projects) != 0 {
		t.Error("project should be deleted")
	}
}

func TestSQLite_ProjectTokens(t *testing.T) {
	store := newTestSQLite(t)
	ctx := context.Background()

	p := &model.Project{Name: "Proj"}
	store.CreateProject(ctx, p)

	tk := &model.ProjectToken{ProjectID: p.ID, Token: "abc123", Label: "v1", Active: true}
	if err := store.CreateProjectToken(ctx, tk); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetProjectTokenByToken(ctx, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got.Label != "v1" {
		t.Errorf("expected v1, got %s", got.Label)
	}

	tokens, err := store.ListProjectTokens(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 {
		t.Errorf("expected 1 token, got %d", len(tokens))
	}

	now := time.Now()
	tk.Active = false
	tk.DisabledAt = &now
	if err := store.UpdateProjectToken(ctx, tk); err != nil {
		t.Fatal(err)
	}

	got2, _ := store.GetProjectTokenByToken(ctx, "abc123")
	if got2.Active {
		t.Error("token should be disabled")
	}

	if err := store.DeleteProjectToken(ctx, tk.ID); err != nil {
		t.Fatal(err)
	}
	tokens, _ = store.ListProjectTokens(ctx, p.ID)
	if len(tokens) != 0 {
		t.Error("token should be deleted")
	}
}

func TestSQLite_DevicesAndUsers(t *testing.T) {
	store := newTestSQLite(t)
	ctx := context.Background()

	p := &model.Project{Name: "Proj"}
	store.CreateProject(ctx, p)

	device := &model.Device{
		ID: "device-123", ProjectID: p.ID, Platform: "ios",
		OSVersion: "18.0", AppVersion: "1.0", Locale: "en_US",
	}
	if err := store.UpsertDevice(ctx, device); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetDevice(ctx, "device-123")
	if err != nil {
		t.Fatal(err)
	}
	if got.Platform != "ios" {
		t.Errorf("expected ios, got %s", got.Platform)
	}

	// Upsert again should update
	device.AppVersion = "2.0"
	if err := store.UpsertDevice(ctx, device); err != nil {
		t.Fatal(err)
	}

	user, err := store.GetOrCreateAppUser(ctx, p.ID, "sha256-hash")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID == 0 {
		t.Error("user ID should be set")
	}

	// Get same user again
	user2, err := store.GetOrCreateAppUser(ctx, p.ID, "sha256-hash")
	if err != nil {
		t.Fatal(err)
	}
	if user2.ID != user.ID {
		t.Error("should return same user")
	}

	if err := store.LinkDeviceToUser(ctx, "device-123", user.ID); err != nil {
		t.Fatal(err)
	}

	uid, err := store.GetDeviceUserID(ctx, "device-123")
	if err != nil {
		t.Fatal(err)
	}
	if uid == nil || *uid != user.ID {
		t.Error("device should be linked to user")
	}
}

func TestSQLite_Events(t *testing.T) {
	store := newTestSQLite(t)
	ctx := context.Background()

	p := &model.Project{Name: "Proj"}
	store.CreateProject(ctx, p)

	device := &model.Device{ID: "dev-1", ProjectID: p.ID, Platform: "android"}
	store.UpsertDevice(ctx, device)

	event := &model.Event{
		ID: "evt-1", ProjectID: p.ID, DeviceID: "dev-1",
		Name: "click", Timestamp: "2026-04-01T10:00:00+03:00",
		IPAddress: "1.2.3.4", Country: "US", City: "NYC",
	}
	inserted, err := store.InsertEvent(ctx, event)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Error("event should be inserted")
	}

	// Dedup: same ID should not insert
	inserted2, err := store.InsertEvent(ctx, event)
	if err != nil {
		t.Fatal(err)
	}
	if inserted2 {
		t.Error("duplicate event should not be inserted")
	}

	// Properties
	s := "USD"
	f := 29.99
	props := []model.EventProperty{
		{EventID: "evt-1", Key: "currency", ValueType: "string", ValueString: &s},
		{EventID: "evt-1", Key: "amount", ValueType: "float", ValueFloat: &f},
	}
	if err := store.InsertEventProperties(ctx, props); err != nil {
		t.Fatal(err)
	}
}

func TestSQLite_EventCleanup(t *testing.T) {
	store := newTestSQLite(t)
	ctx := context.Background()

	p := &model.Project{Name: "Proj", RetentionDays: 7}
	store.CreateProject(ctx, p)

	device := &model.Device{ID: "dev-1", ProjectID: p.ID, Platform: "ios"}
	store.UpsertDevice(ctx, device)

	event := &model.Event{
		ID: "evt-old", ProjectID: p.ID, DeviceID: "dev-1",
		Name: "old_event", Timestamp: "2026-01-01T00:00:00Z",
	}
	store.InsertEvent(ctx, event)

	deleted, err := store.CleanupEvents(ctx, p.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	projects, err := store.GetProjectsWithRetention(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 {
		t.Errorf("expected 1 project with retention, got %d", len(projects))
	}
}

func TestSQLite_Analytics(t *testing.T) {
	store := newTestSQLite(t)
	ctx := context.Background()

	p := &model.Project{Name: "Proj"}
	store.CreateProject(ctx, p)

	device := &model.Device{ID: "dev-1", ProjectID: p.ID, Platform: "ios"}
	store.UpsertDevice(ctx, device)

	for i := 0; i < 5; i++ {
		event := &model.Event{
			ID: fmt.Sprintf("evt-%d", i), ProjectID: p.ID, DeviceID: "dev-1",
			Name: "page_view", Timestamp: "2026-04-01T10:00:00Z",
			Country: "US",
		}
		store.InsertEvent(ctx, event)
	}

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	overview, err := store.GetOverview(ctx, p.ID, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if overview.TotalEvents != 5 {
		t.Errorf("expected 5 events, got %d", overview.TotalEvents)
	}
	if overview.ActiveDevices != 1 {
		t.Errorf("expected 1 device, got %d", overview.ActiveDevices)
	}

	events, err := store.GetEventsAnalytics(ctx, p.ID, from, to, "day", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events.Events) != 1 {
		t.Errorf("expected 1 event type, got %d", len(events.Events))
	}

	devices, err := store.GetDevicesAnalytics(ctx, p.ID, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices.Platforms) != 1 {
		t.Errorf("expected 1 platform, got %d", len(devices.Platforms))
	}

	geo, err := store.GetGeoAnalytics(ctx, p.ID, from, to, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(geo.Countries) == 0 {
		t.Error("expected at least 1 country")
	}
}
