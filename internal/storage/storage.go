package storage

import (
	"context"
	"time"

	"github.com/oxisoft/oximetric/internal/model"
)

type Store interface {
	Close() error
	Ping(ctx context.Context) error

	// Console users
	CreateConsoleUser(ctx context.Context, user *model.ConsoleUser) error
	GetConsoleUserByID(ctx context.Context, id int) (*model.ConsoleUser, error)
	GetConsoleUserByLogin(ctx context.Context, login string) (*model.ConsoleUser, error)
	UpdateConsoleUser(ctx context.Context, user *model.ConsoleUser) error
	DeleteConsoleUser(ctx context.Context, id int) error
	ListConsoleUsers(ctx context.Context) ([]model.ConsoleUser, error)

	// Projects
	CreateProject(ctx context.Context, project *model.Project) error
	GetProject(ctx context.Context, id int) (*model.Project, error)
	UpdateProject(ctx context.Context, project *model.Project) error
	DeleteProject(ctx context.Context, id int) error
	ListProjects(ctx context.Context) ([]model.Project, error)

	// Project tokens
	CreateProjectToken(ctx context.Context, token *model.ProjectToken) error
	GetProjectTokenByToken(ctx context.Context, token string) (*model.ProjectToken, error)
	ListProjectTokens(ctx context.Context, projectID int) ([]model.ProjectToken, error)
	UpdateProjectToken(ctx context.Context, token *model.ProjectToken) error
	DeleteProjectToken(ctx context.Context, id int) error

	// Devices
	UpsertDevice(ctx context.Context, device *model.Device) error
	GetDevice(ctx context.Context, id string) (*model.Device, error)
	LinkDeviceToUser(ctx context.Context, deviceID string, userID int) error

	// App users
	GetOrCreateAppUser(ctx context.Context, projectID int, anonymousID string) (*model.AppUser, error)

	// Events
	InsertEvent(ctx context.Context, event *model.Event) (bool, error)
	InsertEventProperties(ctx context.Context, props []model.EventProperty) error
	GetDeviceUserID(ctx context.Context, deviceID string) (*int, error)
	CleanupEvents(ctx context.Context, projectID int, before time.Time) (int64, error)

	// Analytics
	GetOverview(ctx context.Context, projectID int, from, to time.Time) (*model.AnalyticsOverview, error)
	GetEventsAnalytics(ctx context.Context, projectID int, from, to time.Time, interval string, limit, offset int) (*model.EventsAnalytics, error)
	GetEventProperties(ctx context.Context, projectID int, eventName string, from, to time.Time, limit, offset int) (*model.PropertyAnalytics, error)
	GetDevicesAnalytics(ctx context.Context, projectID int, from, to time.Time) (*model.DevicesAnalytics, error)
	GetGeoAnalytics(ctx context.Context, projectID int, from, to time.Time, limit, offset int) (*model.GeoAnalytics, error)
	GetUsersAnalytics(ctx context.Context, projectID int, from, to time.Time, interval string) (*model.UsersAnalytics, error)
	GetRetentionAnalytics(ctx context.Context, projectID int, from, to time.Time) (*model.RetentionAnalytics, error)

	// Retention cleanup
	GetProjectsWithRetention(ctx context.Context) ([]model.Project, error)
}
