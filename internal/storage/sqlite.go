package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/oxisoft/oximetric/internal/model"
	_ "modernc.org/sqlite"
)

//go:embed migrations/sqlite/*.sql
var sqliteMigrations embed.FS

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLite(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dsn+"?_pragma=foreign_keys(1)&_pragma=journal_mode(wal)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate sqlite: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	data, err := sqliteMigrations.ReadFile("migrations/sqlite/001_initial.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	_, err = s.db.Exec(string(data))
	return err
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// Console users

func (s *SQLiteStore) CreateConsoleUser(ctx context.Context, user *model.ConsoleUser) error {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO console_users (username, email, password_hash, role, totp_secret, totp_enabled) VALUES (?, ?, ?, ?, ?, ?)`,
		user.Username, user.Email, user.PasswordHash, user.Role, user.TOTPSecret, user.TOTPEnabled,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = int(id)
	return nil
}

func (s *SQLiteStore) GetConsoleUserByID(ctx context.Context, id int) (*model.ConsoleUser, error) {
	var u model.ConsoleUser
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, password_hash, role, totp_secret, totp_enabled, created_at FROM console_users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *SQLiteStore) GetConsoleUserByLogin(ctx context.Context, login string) (*model.ConsoleUser, error) {
	var u model.ConsoleUser
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, password_hash, role, totp_secret, totp_enabled, created_at FROM console_users WHERE username = ? OR email = ?`, login, login,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *SQLiteStore) UpdateConsoleUser(ctx context.Context, user *model.ConsoleUser) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE console_users SET username = ?, email = ?, password_hash = ?, role = ?, totp_secret = ?, totp_enabled = ? WHERE id = ?`,
		user.Username, user.Email, user.PasswordHash, user.Role, user.TOTPSecret, user.TOTPEnabled, user.ID,
	)
	return err
}

func (s *SQLiteStore) DeleteConsoleUser(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM console_users WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) ListConsoleUsers(ctx context.Context) ([]model.ConsoleUser, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, username, email, role, totp_enabled, created_at FROM console_users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []model.ConsoleUser
	for rows.Next() {
		var u model.ConsoleUser
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.TOTPEnabled, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Projects

func (s *SQLiteStore) CreateProject(ctx context.Context, project *model.Project) error {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO projects (name, retention_days) VALUES (?, ?)`,
		project.Name, project.RetentionDays,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	project.ID = int(id)
	return nil
}

func (s *SQLiteStore) GetProject(ctx context.Context, id int) (*model.Project, error) {
	var p model.Project
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, retention_days, created_at FROM projects WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.RetentionDays, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *SQLiteStore) UpdateProject(ctx context.Context, project *model.Project) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE projects SET name = ?, retention_days = ? WHERE id = ?`,
		project.Name, project.RetentionDays, project.ID,
	)
	return err
}

func (s *SQLiteStore) DeleteProject(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) ListProjects(ctx context.Context) ([]model.Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, retention_days, created_at FROM projects ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projects []model.Project
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.RetentionDays, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// Project tokens

func (s *SQLiteStore) CreateProjectToken(ctx context.Context, token *model.ProjectToken) error {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO project_tokens (project_id, token, label, active) VALUES (?, ?, ?, ?)`,
		token.ProjectID, token.Token, token.Label, token.Active,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	token.ID = int(id)
	return nil
}

func (s *SQLiteStore) GetProjectTokenByToken(ctx context.Context, token string) (*model.ProjectToken, error) {
	var t model.ProjectToken
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, token, label, active, created_at, disabled_at FROM project_tokens WHERE token = ?`, token,
	).Scan(&t.ID, &t.ProjectID, &t.Token, &t.Label, &t.Active, &t.CreatedAt, &t.DisabledAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *SQLiteStore) ListProjectTokens(ctx context.Context, projectID int) ([]model.ProjectToken, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, token, label, active, created_at, disabled_at FROM project_tokens WHERE project_id = ? ORDER BY created_at`, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tokens []model.ProjectToken
	for rows.Next() {
		var t model.ProjectToken
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Token, &t.Label, &t.Active, &t.CreatedAt, &t.DisabledAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func (s *SQLiteStore) UpdateProjectToken(ctx context.Context, token *model.ProjectToken) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE project_tokens SET label = ?, active = ?, disabled_at = ? WHERE id = ?`,
		token.Label, token.Active, token.DisabledAt, token.ID,
	)
	return err
}

func (s *SQLiteStore) DeleteProjectToken(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM project_tokens WHERE id = ?`, id)
	return err
}

// Devices

func (s *SQLiteStore) UpsertDevice(ctx context.Context, device *model.Device) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO devices (id, project_id, platform, os_version, app_version, locale, first_seen_at, last_seen_at)
		 VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		 ON CONFLICT(id) DO UPDATE SET
		   os_version = excluded.os_version,
		   app_version = excluded.app_version,
		   locale = excluded.locale,
		   last_seen_at = CURRENT_TIMESTAMP`,
		device.ID, device.ProjectID, device.Platform, device.OSVersion, device.AppVersion, device.Locale,
	)
	return err
}

func (s *SQLiteStore) GetDevice(ctx context.Context, id string) (*model.Device, error) {
	var d model.Device
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, user_id, platform, COALESCE(os_version,''), COALESCE(app_version,''), COALESCE(locale,''), first_seen_at, last_seen_at FROM devices WHERE id = ?`, id,
	).Scan(&d.ID, &d.ProjectID, &d.UserID, &d.Platform, &d.OSVersion, &d.AppVersion, &d.Locale, &d.FirstSeenAt, &d.LastSeenAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *SQLiteStore) LinkDeviceToUser(ctx context.Context, deviceID string, userID int) error {
	_, err := s.db.ExecContext(ctx, `UPDATE devices SET user_id = ? WHERE id = ?`, userID, deviceID)
	return err
}

// App users

func (s *SQLiteStore) GetOrCreateAppUser(ctx context.Context, projectID int, anonymousID string) (*model.AppUser, error) {
	var u model.AppUser
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, anonymous_id, created_at FROM app_users WHERE project_id = ? AND anonymous_id = ?`,
		projectID, anonymousID,
	).Scan(&u.ID, &u.ProjectID, &u.AnonymousID, &u.CreatedAt)
	if err == nil {
		return &u, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO app_users (project_id, anonymous_id) VALUES (?, ?)`,
		projectID, anonymousID,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	u.ID = int(id)
	u.ProjectID = projectID
	u.AnonymousID = anonymousID
	return &u, nil
}

// Events

func (s *SQLiteStore) InsertEvent(ctx context.Context, event *model.Event) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO events (id, project_id, device_id, user_id, name, timestamp, ip_address, country, city)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.ProjectID, event.DeviceID, event.UserID, event.Name, event.Timestamp,
		nullStr(event.IPAddress), nullStr(event.Country), nullStr(event.City),
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SQLiteStore) InsertEventProperties(ctx context.Context, props []model.EventProperty) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO event_properties (event_id, key, value_type, value_string, value_int, value_float, value_bool, value_datetime) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range props {
		if _, err := stmt.ExecContext(ctx, p.EventID, p.Key, p.ValueType, p.ValueString, p.ValueInt, p.ValueFloat, p.ValueBool, p.ValueDatetime); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) GetDeviceUserID(ctx context.Context, deviceID string) (*int, error) {
	var userID *int
	err := s.db.QueryRowContext(ctx, `SELECT user_id FROM devices WHERE id = ?`, deviceID).Scan(&userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return userID, err
}

func (s *SQLiteStore) CleanupEvents(ctx context.Context, projectID int, before time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM events WHERE project_id = ? AND created_at < ?`, projectID, before,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) GetProjectsWithRetention(ctx context.Context) ([]model.Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, retention_days, created_at FROM projects WHERE retention_days > 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projects []model.Project
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.RetentionDays, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// Analytics

func (s *SQLiteStore) GetOverview(ctx context.Context, projectID int, from, to time.Time) (*model.AnalyticsOverview, error) {
	var overview model.AnalyticsOverview

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT device_id) FROM events WHERE project_id = ? AND created_at BETWEEN ? AND ?`,
		projectID, from, to,
	).Scan(&overview.ActiveDevices)

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM events WHERE project_id = ? AND created_at BETWEEN ? AND ?`,
		projectID, from, to,
	).Scan(&overview.TotalEvents)

	rows, err := s.db.QueryContext(ctx,
		`SELECT name, COUNT(*) as cnt FROM events WHERE project_id = ? AND created_at BETWEEN ? AND ? GROUP BY name ORDER BY cnt DESC LIMIT 10`,
		projectID, from, to,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var e model.EventCount
			rows.Scan(&e.Name, &e.Count)
			overview.TopEvents = append(overview.TopEvents, e)
		}
	}

	rows2, err := s.db.QueryContext(ctx,
		`SELECT COALESCE(country,'Unknown'), COUNT(*) as cnt FROM events WHERE project_id = ? AND created_at BETWEEN ? AND ? GROUP BY country ORDER BY cnt DESC LIMIT 10`,
		projectID, from, to,
	)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var c model.CountryCount
			rows2.Scan(&c.Country, &c.Count)
			overview.TopCountries = append(overview.TopCountries, c)
		}
	}

	return &overview, nil
}

func (s *SQLiteStore) GetEventsAnalytics(ctx context.Context, projectID int, from, to time.Time, interval string, limit, offset int) (*model.EventsAnalytics, error) {
	result := &model.EventsAnalytics{}

	rows, err := s.db.QueryContext(ctx,
		`SELECT name, COUNT(*) as cnt FROM events WHERE project_id = ? AND created_at BETWEEN ? AND ? GROUP BY name ORDER BY cnt DESC LIMIT ? OFFSET ?`,
		projectID, from, to, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e model.EventCount
		rows.Scan(&e.Name, &e.Count)
		result.Events = append(result.Events, e)
	}

	format := sqliteDateFormat(interval)
	rows2, err := s.db.QueryContext(ctx,
		`SELECT strftime(?, created_at) as period, COUNT(*) as cnt FROM events WHERE project_id = ? AND created_at BETWEEN ? AND ? GROUP BY period ORDER BY period`,
		format, projectID, from, to,
	)
	if err != nil {
		return result, nil
	}
	defer rows2.Close()
	for rows2.Next() {
		var t model.TimeSeriesPoint
		rows2.Scan(&t.Timestamp, &t.Count)
		result.TimeSeries = append(result.TimeSeries, t)
	}

	return result, nil
}

func (s *SQLiteStore) GetEventProperties(ctx context.Context, projectID int, eventName string, from, to time.Time, limit, offset int) (*model.PropertyAnalytics, error) {
	result := &model.PropertyAnalytics{}

	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT ep.key, ep.value_type
		 FROM event_properties ep
		 JOIN events e ON e.id = ep.event_id
		 WHERE e.project_id = ? AND e.name = ? AND e.created_at BETWEEN ? AND ?
		 LIMIT ? OFFSET ?`,
		projectID, eventName, from, to, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pb model.PropertyBreakdown
		rows.Scan(&pb.Key, &pb.Type)

		valCol := propertyValueColumn(pb.Type)
		vrows, err := s.db.QueryContext(ctx,
			fmt.Sprintf(`SELECT %s, COUNT(*) as cnt
			 FROM event_properties ep
			 JOIN events e ON e.id = ep.event_id
			 WHERE e.project_id = ? AND e.name = ? AND ep.key = ? AND e.created_at BETWEEN ? AND ?
			 GROUP BY %s ORDER BY cnt DESC LIMIT 20`, valCol, valCol),
			projectID, eventName, pb.Key, from, to,
		)
		if err == nil {
			for vrows.Next() {
				var pv model.PropertyValue
				vrows.Scan(&pv.Value, &pv.Count)
				pb.Values = append(pb.Values, pv)
			}
			vrows.Close()
		}
		result.Properties = append(result.Properties, pb)
	}

	return result, nil
}

func (s *SQLiteStore) GetDevicesAnalytics(ctx context.Context, projectID int, from, to time.Time) (*model.DevicesAnalytics, error) {
	result := &model.DevicesAnalytics{}
	rows, err := s.db.QueryContext(ctx,
		`SELECT platform, COUNT(*) as cnt FROM devices WHERE project_id = ? AND last_seen_at BETWEEN ? AND ? GROUP BY platform ORDER BY cnt DESC`,
		projectID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p model.PlatformCount
		rows.Scan(&p.Platform, &p.Count)
		result.Platforms = append(result.Platforms, p)
	}
	return result, nil
}

func (s *SQLiteStore) GetGeoAnalytics(ctx context.Context, projectID int, from, to time.Time, limit, offset int) (*model.GeoAnalytics, error) {
	result := &model.GeoAnalytics{}

	rows, err := s.db.QueryContext(ctx,
		`SELECT COALESCE(country,'Unknown'), COUNT(*) as cnt FROM events WHERE project_id = ? AND created_at BETWEEN ? AND ? GROUP BY country ORDER BY cnt DESC LIMIT ? OFFSET ?`,
		projectID, from, to, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c model.CountryCount
		rows.Scan(&c.Country, &c.Count)
		result.Countries = append(result.Countries, c)
	}

	rows2, err := s.db.QueryContext(ctx,
		`SELECT COALESCE(city,'Unknown'), COALESCE(country,'Unknown'), COUNT(*) as cnt FROM events WHERE project_id = ? AND created_at BETWEEN ? AND ? GROUP BY city, country ORDER BY cnt DESC LIMIT ? OFFSET ?`,
		projectID, from, to, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var c model.CityCount
		rows2.Scan(&c.City, &c.Country, &c.Count)
		result.Cities = append(result.Cities, c)
	}

	return result, nil
}

func (s *SQLiteStore) GetUsersAnalytics(ctx context.Context, projectID int, from, to time.Time, interval string) (*model.UsersAnalytics, error) {
	result := &model.UsersAnalytics{}

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM app_users WHERE project_id = ?`, projectID,
	).Scan(&result.Total)

	format := sqliteDateFormat(interval)
	rows, err := s.db.QueryContext(ctx,
		`SELECT strftime(?, created_at) as period, COUNT(*) as cnt FROM app_users WHERE project_id = ? AND created_at BETWEEN ? AND ? GROUP BY period ORDER BY period`,
		format, projectID, from, to,
	)
	if err != nil {
		return result, nil
	}
	defer rows.Close()
	for rows.Next() {
		var t model.TimeSeriesPoint
		rows.Scan(&t.Timestamp, &t.Count)
		result.TimeSeries = append(result.TimeSeries, t)
	}
	return result, nil
}

func (s *SQLiteStore) GetRetentionAnalytics(ctx context.Context, projectID int, from, to time.Time) (*model.RetentionAnalytics, error) {
	return &model.RetentionAnalytics{Cohorts: []model.RetentionCohort{}}, nil
}

// Helpers

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func sqliteDateFormat(interval string) string {
	switch interval {
	case "hour":
		return "%Y-%m-%d %H:00"
	case "week":
		return "%Y-W%W"
	case "month":
		return "%Y-%m"
	default:
		return "%Y-%m-%d"
	}
}

func propertyValueColumn(valueType string) string {
	switch valueType {
	case "int":
		return "value_int"
	case "float":
		return "value_float"
	case "bool":
		return "value_bool"
	case "datetime":
		return "value_datetime"
	default:
		return "value_string"
	}
}
