package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/oxisoft/oximetric/internal/model"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, model.ErrorResponse{
		Error: model.ErrorDetail{Code: code, Message: message},
	})
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func pathInt(r *http.Request, name string) (int, error) {
	return strconv.Atoi(r.PathValue(name))
}

type analyticsParams struct {
	From     time.Time
	To       time.Time
	Interval string
	Limit    int
	Offset   int
}

func parseAnalyticsParams(r *http.Request) analyticsParams {
	p := analyticsParams{
		From:     time.Now().AddDate(0, 0, -30),
		To:       time.Now(),
		Interval: "day",
		Limit:    50,
		Offset:   0,
	}
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			p.From = t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			p.To = t
		}
	}
	if v := r.URL.Query().Get("interval"); v != "" {
		p.Interval = v
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			p.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			p.Offset = n
		}
	}
	return p
}
