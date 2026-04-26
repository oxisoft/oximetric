package model

import "time"

type Project struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	RetentionDays int       `json:"retention_days"`
	CreatedAt     time.Time `json:"created_at"`
}

type ProjectToken struct {
	ID             int        `json:"id"`
	ProjectID      int        `json:"project_id"`
	Token          string     `json:"token"`
	Label          string     `json:"label"`
	Active         bool       `json:"active"`
	AllowedOrigins []string   `json:"allowed_origins"`
	CreatedAt      time.Time  `json:"created_at"`
	DisabledAt     *time.Time `json:"disabled_at,omitempty"`
}

type ConsoleUser struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        *string   `json:"email,omitempty"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	TOTPSecret *string `json:"-"`
	TOTPEnabled  bool      `json:"totp_enabled"`
	CreatedAt    time.Time `json:"created_at"`
}

type AppUser struct {
	ID          int       `json:"id"`
	ProjectID   int       `json:"project_id"`
	AnonymousID string    `json:"anonymous_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type Device struct {
	ID          string    `json:"id"`
	ProjectID   int       `json:"project_id"`
	UserID      *int      `json:"user_id,omitempty"`
	Platform    string    `json:"platform"`
	OSVersion   string    `json:"os_version"`
	AppVersion  string    `json:"app_version"`
	Locale      string    `json:"locale"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

type Event struct {
	ID        string    `json:"id"`
	ProjectID int       `json:"project_id"`
	DeviceID  string    `json:"device_id"`
	UserID    *int      `json:"user_id,omitempty"`
	Name      string    `json:"name"`
	Timestamp string    `json:"timestamp"`
	IPAddress string    `json:"ip_address,omitempty"`
	Country   string    `json:"country,omitempty"`
	City      string    `json:"city,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type EventProperty struct {
	ID            int        `json:"id"`
	EventID       string     `json:"event_id"`
	Key           string     `json:"key"`
	ValueType     string     `json:"value_type"`
	ValueString   *string    `json:"value_string,omitempty"`
	ValueInt      *int64     `json:"value_int,omitempty"`
	ValueFloat    *float64   `json:"value_float,omitempty"`
	ValueBool     *bool      `json:"value_bool,omitempty"`
	ValueDatetime *time.Time `json:"value_datetime,omitempty"`
}

// API request/response types

type TrackRequest struct {
	DeviceID string              `json:"device_id"`
	UserID   *int                `json:"user_id,omitempty"`
	Events   []TrackEventRequest `json:"events"`
}

type TrackEventRequest struct {
	ID         string                       `json:"id"`
	Name       string                       `json:"name"`
	Timestamp  string                       `json:"timestamp"`
	Properties map[string]TrackPropertyValue `json:"properties"`
}

type TrackPropertyValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type TrackResponse struct {
	Accepted int `json:"accepted"`
}

type DeviceRequest struct {
	DeviceID   string `json:"device_id"`
	Platform   string `json:"platform"`
	OSVersion  string `json:"os_version"`
	AppVersion string `json:"app_version"`
	Locale     string `json:"locale"`
}

type IdentifyRequest struct {
	DeviceID    string `json:"device_id"`
	AnonymousID string `json:"anonymous_id"`
}

type IdentifyResponse struct {
	UserID int `json:"user_id"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	TOTPCode string `json:"totp_code,omitempty"`
}

type LoginResponse struct {
	Token        string       `json:"token,omitempty"`
	User         *ConsoleUser `json:"user,omitempty"`
	TOTPRequired bool         `json:"totp_required,omitempty"`
}

type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type TOTPSetupResponse struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

type TOTPEnableRequest struct {
	Code     string `json:"code"`
	Password string `json:"password"`
}

type TOTPDisableRequest struct {
	Password string `json:"password"`
}

type CreateProjectRequest struct {
	Name string `json:"name"`
}

type UpdateProjectRequest struct {
	Name          *string `json:"name,omitempty"`
	RetentionDays *int    `json:"retention_days,omitempty"`
}

type CreateTokenRequest struct {
	Label          string   `json:"label"`
	AllowedOrigins []string `json:"allowed_origins"`
}

type UpdateTokenRequest struct {
	Label          *string   `json:"label,omitempty"`
	AllowedOrigins *[]string `json:"allowed_origins,omitempty"`
}

type CreateUserRequest struct {
	Username string  `json:"username"`
	Email    *string `json:"email,omitempty"`
	Password string  `json:"password"`
	Role     string  `json:"role"`
}

type UpdateUserRequest struct {
	Username *string `json:"username,omitempty"`
	Email    *string `json:"email,omitempty"`
	Role     *string `json:"role,omitempty"`
}

type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	GeoIP    string `json:"geoip"`
	Version  string `json:"version"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Analytics response types

type AnalyticsOverview struct {
	ActiveDevices int              `json:"active_devices"`
	TotalEvents   int              `json:"total_events"`
	TopEvents     []EventCount     `json:"top_events"`
	TopCountries  []CountryCount   `json:"top_countries"`
}

type EventCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type CountryCount struct {
	Country string `json:"country"`
	Count   int    `json:"count"`
}

type EventsAnalytics struct {
	Events     []EventCount     `json:"events"`
	TimeSeries []TimeSeriesPoint `json:"time_series"`
}

type TimeSeriesPoint struct {
	Timestamp string `json:"timestamp"`
	Count     int    `json:"count"`
}

type PropertyAnalytics struct {
	Properties []PropertyBreakdown `json:"properties"`
}

type PropertyBreakdown struct {
	Key    string          `json:"key"`
	Type   string          `json:"type"`
	Values []PropertyValue `json:"values"`
}

type PropertyValue struct {
	Value interface{} `json:"value"`
	Count int         `json:"count"`
}

type DevicesAnalytics struct {
	Platforms []PlatformCount `json:"platforms"`
}

type PlatformCount struct {
	Platform string `json:"platform"`
	Count    int    `json:"count"`
}

type GeoAnalytics struct {
	Countries []CountryCount `json:"countries"`
	Cities    []CityCount    `json:"cities"`
}

type CityCount struct {
	City    string `json:"city"`
	Country string `json:"country"`
	Count   int    `json:"count"`
}

type UsersAnalytics struct {
	TimeSeries []TimeSeriesPoint `json:"time_series"`
	Total      int               `json:"total"`
}

type RetentionAnalytics struct {
	Cohorts []RetentionCohort `json:"cohorts"`
}

type RetentionCohort struct {
	Period     string    `json:"period"`
	Users      int       `json:"users"`
	Retention  []float64 `json:"retention"`
}
