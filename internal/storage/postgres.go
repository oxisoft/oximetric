package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/oxisoft/oximetric/internal/model"
	_ "github.com/lib/pq"
)

//go:embed migrations/postgres/*.sql
var postgresMigrations embed.FS

type PostgresStore struct {
	db *sql.DB
}

func NewPostgres(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	s := &PostgresStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate postgres: %w", err)
	}
	return s, nil
}

func (s *PostgresStore) migrate() error {
	data, err := postgresMigrations.ReadFile("migrations/postgres/001_initial.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	_, err = s.db.Exec(string(data))
	return err
}

func (s *PostgresStore) Close() error { return s.db.Close() }

func (s *PostgresStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// Console users

func (s *PostgresStore) CreateConsoleUser(ctx context.Context, user *model.ConsoleUser) error {
	return s.db.QueryRowContext(ctx,
		`INSERT INTO console_users (username, email, password_hash, role, totp_secret, totp_enabled) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		user.Username, user.Email, user.PasswordHash, user.Role, user.TOTPSecret, user.TOTPEnabled,
	).Scan(&user.ID)
}

func (s *PostgresStore) GetConsoleUserByID(ctx context.Context, id int) (*model.ConsoleUser, error) {
	var u model.ConsoleUser
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, password_hash, role, totp_secret, totp_enabled, created_at FROM console_users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *PostgresStore) GetConsoleUserByLogin(ctx context.Context, login string) (*model.ConsoleUser, error) {
	var u model.ConsoleUser
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, password_hash, role, totp_secret, totp_enabled, created_at FROM console_users WHERE username = $1 OR email = $1`, login,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *PostgresStore) UpdateConsoleUser(ctx context.Context, user *model.ConsoleUser) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE console_users SET username = $1, email = $2, password_hash = $3, role = $4, totp_secret = $5, totp_enabled = $6 WHERE id = $7`,
		user.Username, user.Email, user.PasswordHash, user.Role, user.TOTPSecret, user.TOTPEnabled, user.ID,
	)
	return err
}

func (s *PostgresStore) DeleteConsoleUser(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM console_users WHERE id = $1`, id)
	return err
}

func (s *PostgresStore) ListConsoleUsers(ctx context.Context) ([]model.ConsoleUser, error) {
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

func (s *PostgresStore) CreateProject(ctx context.Context, project *model.Project) error {
	return s.db.QueryRowContext(ctx,
		`INSERT INTO projects (name, retention_days) VALUES ($1, $2) RETURNING id`,
		project.Name, project.RetentionDays,
	).Scan(&project.ID)
}

func (s *PostgresStore) GetProject(ctx context.Context, id int) (*model.Project, error) {
	var p model.Project
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, retention_days, created_at FROM projects WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.RetentionDays, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *PostgresStore) UpdateProject(ctx context.Context, project *model.Project) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE projects SET name = $1, retention_days = $2 WHERE id = $3`,
		project.Name, project.RetentionDays, project.ID,
	)
	return err
}

func (s *PostgresStore) DeleteProject(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
	return err
}

func (s *PostgresStore) ListProjects(ctx context.Context) ([]model.Project, error) {
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

func (s *PostgresStore) CreateProjectToken(ctx context.Context, token *model.ProjectToken) error {
	return s.db.QueryRowContext(ctx,
		`INSERT INTO project_tokens (project_id, token, label, active, allowed_origins) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		token.ProjectID, token.Token, token.Label, token.Active, encodeOrigins(token.AllowedOrigins),
	).Scan(&token.ID)
}

func (s *PostgresStore) GetProjectTokenByToken(ctx context.Context, token string) (*model.ProjectToken, error) {
	var t model.ProjectToken
	var origins string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, token, label, active, allowed_origins, created_at, disabled_at FROM project_tokens WHERE token = $1`, token,
	).Scan(&t.ID, &t.ProjectID, &t.Token, &t.Label, &t.Active, &origins, &t.CreatedAt, &t.DisabledAt)
	if err != nil {
		return nil, err
	}
	t.AllowedOrigins = decodeOrigins(origins)
	return &t, nil
}

func (s *PostgresStore) ListProjectTokens(ctx context.Context, projectID int) ([]model.ProjectToken, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, token, label, active, allowed_origins, created_at, disabled_at FROM project_tokens WHERE project_id = $1 ORDER BY created_at`, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tokens []model.ProjectToken
	for rows.Next() {
		var t model.ProjectToken
		var origins string
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Token, &t.Label, &t.Active, &origins, &t.CreatedAt, &t.DisabledAt); err != nil {
			return nil, err
		}
		t.AllowedOrigins = decodeOrigins(origins)
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func (s *PostgresStore) UpdateProjectToken(ctx context.Context, token *model.ProjectToken) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE project_tokens SET label = $1, active = $2, allowed_origins = $3, disabled_at = $4 WHERE id = $5`,
		token.Label, token.Active, encodeOrigins(token.AllowedOrigins), token.DisabledAt, token.ID,
	)
	return err
}

func (s *PostgresStore) DeleteProjectToken(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM project_tokens WHERE id = $1`, id)
	return err
}

// Devices

func (s *PostgresStore) UpsertDevice(ctx context.Context, device *model.Device) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO devices (id, project_id, platform, os_version, app_version, locale, first_seen_at, last_seen_at)
		 VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		 ON CONFLICT(id) DO UPDATE SET
		   os_version = EXCLUDED.os_version,
		   app_version = EXCLUDED.app_version,
		   locale = EXCLUDED.locale,
		   last_seen_at = CURRENT_TIMESTAMP`,
		device.ID, device.ProjectID, device.Platform, device.OSVersion, device.AppVersion, device.Locale,
	)
	return err
}

func (s *PostgresStore) GetDevice(ctx context.Context, id string) (*model.Device, error) {
	var d model.Device
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, user_id, platform, COALESCE(os_version,''), COALESCE(app_version,''), COALESCE(locale,''), first_seen_at, last_seen_at FROM devices WHERE id = $1`, id,
	).Scan(&d.ID, &d.ProjectID, &d.UserID, &d.Platform, &d.OSVersion, &d.AppVersion, &d.Locale, &d.FirstSeenAt, &d.LastSeenAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *PostgresStore) LinkDeviceToUser(ctx context.Context, deviceID string, userID int) error {
	_, err := s.db.ExecContext(ctx, `UPDATE devices SET user_id = $1 WHERE id = $2`, userID, deviceID)
	return err
}

// App users

func (s *PostgresStore) GetOrCreateAppUser(ctx context.Context, projectID int, anonymousID string) (*model.AppUser, error) {
	var u model.AppUser
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, anonymous_id, created_at FROM app_users WHERE project_id = $1 AND anonymous_id = $2`,
		projectID, anonymousID,
	).Scan(&u.ID, &u.ProjectID, &u.AnonymousID, &u.CreatedAt)
	if err == nil {
		return &u, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	err = s.db.QueryRowContext(ctx,
		`INSERT INTO app_users (project_id, anonymous_id) VALUES ($1, $2) RETURNING id`,
		projectID, anonymousID,
	).Scan(&u.ID)
	if err != nil {
		return nil, err
	}
	u.ProjectID = projectID
	u.AnonymousID = anonymousID
	return &u, nil
}

// Events

func (s *PostgresStore) InsertEvent(ctx context.Context, event *model.Event) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO events (id, project_id, device_id, user_id, name, timestamp, ip_address, country, city)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT(id) DO NOTHING`,
		event.ID, event.ProjectID, event.DeviceID, event.UserID, event.Name, event.Timestamp,
		nullStr(event.IPAddress), nullStr(event.Country), nullStr(event.City),
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *PostgresStore) InsertEventProperties(ctx context.Context, props []model.EventProperty) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO event_properties (event_id, key, value_type, value_string, value_int, value_float, value_bool, value_datetime) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
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

func (s *PostgresStore) GetDeviceUserID(ctx context.Context, deviceID string) (*int, error) {
	var userID *int
	err := s.db.QueryRowContext(ctx, `SELECT user_id FROM devices WHERE id = $1`, deviceID).Scan(&userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return userID, err
}

func (s *PostgresStore) CleanupEvents(ctx context.Context, projectID int, before time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM events WHERE project_id = $1 AND created_at < $2`, projectID, before,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *PostgresStore) GetProjectsWithRetention(ctx context.Context) ([]model.Project, error) {
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

func (s *PostgresStore) GetOverview(ctx context.Context, projectID int, from, to time.Time) (*model.AnalyticsOverview, error) {
	var overview model.AnalyticsOverview

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT device_id) FROM events WHERE project_id = $1 AND created_at BETWEEN $2 AND $3`,
		projectID, from, to,
	).Scan(&overview.ActiveDevices)

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM events WHERE project_id = $1 AND created_at BETWEEN $2 AND $3`,
		projectID, from, to,
	).Scan(&overview.TotalEvents)

	rows, err := s.db.QueryContext(ctx,
		`SELECT name, COUNT(*) as cnt FROM events WHERE project_id = $1 AND created_at BETWEEN $2 AND $3 GROUP BY name ORDER BY cnt DESC LIMIT 10`,
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
		`SELECT COALESCE(country,'Unknown'), COUNT(*) as cnt FROM events WHERE project_id = $1 AND created_at BETWEEN $2 AND $3 GROUP BY country ORDER BY cnt DESC LIMIT 10`,
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

func (s *PostgresStore) GetEventsAnalytics(ctx context.Context, projectID int, from, to time.Time, interval string, limit, offset int) (*model.EventsAnalytics, error) {
	result := &model.EventsAnalytics{}

	rows, err := s.db.QueryContext(ctx,
		`SELECT name, COUNT(*) as cnt FROM events WHERE project_id = $1 AND created_at BETWEEN $2 AND $3 GROUP BY name ORDER BY cnt DESC LIMIT $4 OFFSET $5`,
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

	trunc := pgDateTrunc(interval)
	rows2, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT date_trunc('%s', created_at)::text as period, COUNT(*) as cnt FROM events WHERE project_id = $1 AND created_at BETWEEN $2 AND $3 GROUP BY period ORDER BY period`, trunc),
		projectID, from, to,
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

func (s *PostgresStore) GetEventProperties(ctx context.Context, projectID int, eventName string, from, to time.Time, limit, offset int) (*model.PropertyAnalytics, error) {
	result := &model.PropertyAnalytics{}

	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT ep.key, ep.value_type
		 FROM event_properties ep
		 JOIN events e ON e.id = ep.event_id
		 WHERE e.project_id = $1 AND e.name = $2 AND e.created_at BETWEEN $3 AND $4
		 LIMIT $5 OFFSET $6`,
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
			 WHERE e.project_id = $1 AND e.name = $2 AND ep.key = $3 AND e.created_at BETWEEN $4 AND $5
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

func (s *PostgresStore) GetDevicesAnalytics(ctx context.Context, projectID int, from, to time.Time) (*model.DevicesAnalytics, error) {
	result := &model.DevicesAnalytics{}
	rows, err := s.db.QueryContext(ctx,
		`SELECT platform, COUNT(*) as cnt FROM devices WHERE project_id = $1 AND last_seen_at BETWEEN $2 AND $3 GROUP BY platform ORDER BY cnt DESC`,
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

func (s *PostgresStore) GetGeoAnalytics(ctx context.Context, projectID int, from, to time.Time, limit, offset int) (*model.GeoAnalytics, error) {
	result := &model.GeoAnalytics{}

	rows, err := s.db.QueryContext(ctx,
		`SELECT COALESCE(country,'Unknown'), COUNT(*) as cnt FROM events WHERE project_id = $1 AND created_at BETWEEN $2 AND $3 GROUP BY country ORDER BY cnt DESC LIMIT $4 OFFSET $5`,
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
		`SELECT COALESCE(city,'Unknown'), COALESCE(country,'Unknown'), COUNT(*) as cnt FROM events WHERE project_id = $1 AND created_at BETWEEN $2 AND $3 GROUP BY city, country ORDER BY cnt DESC LIMIT $4 OFFSET $5`,
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

func (s *PostgresStore) GetUsersAnalytics(ctx context.Context, projectID int, from, to time.Time, interval string) (*model.UsersAnalytics, error) {
	result := &model.UsersAnalytics{}

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM app_users WHERE project_id = $1`, projectID,
	).Scan(&result.Total)

	trunc := pgDateTrunc(interval)
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT date_trunc('%s', created_at)::text as period, COUNT(*) as cnt FROM app_users WHERE project_id = $1 AND created_at BETWEEN $2 AND $3 GROUP BY period ORDER BY period`, trunc),
		projectID, from, to,
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

func (s *PostgresStore) GetRetentionAnalytics(ctx context.Context, projectID int, from, to time.Time) (*model.RetentionAnalytics, error) {
	return &model.RetentionAnalytics{Cohorts: []model.RetentionCohort{}}, nil
}

func pgDateTrunc(interval string) string {
	switch interval {
	case "hour":
		return "hour"
	case "week":
		return "week"
	case "month":
		return "month"
	default:
		return "day"
	}
}
