package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

// normalizeOrigins validates and canonicalizes a list of allowed origins.
// Each entry must be either:
//   - a full origin: scheme://host[:port]   (e.g. https://example.com)
//   - a wildcard subdomain pattern: scheme://*.host[:port]
//   - the literal "*" meaning any origin (NOT recommended)
//
// Trailing slashes and paths are stripped. Duplicates and blanks are removed.
func normalizeOrigins(in []string) ([]string, error) {
	seen := make(map[string]bool)
	out := make([]string, 0, len(in))
	for _, raw := range in {
		o := strings.TrimSpace(raw)
		if o == "" {
			continue
		}
		if o == "*" {
			if !seen[o] {
				out = append(out, o)
				seen[o] = true
			}
			continue
		}
		// Wildcard subdomain: replace * with a sentinel for url.Parse.
		check := o
		hasWild := strings.Contains(check, "://*.")
		if hasWild {
			check = strings.Replace(check, "://*.", "://wildcard.", 1)
		}
		u, err := url.Parse(check)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("invalid origin %q: must be scheme://host[:port]", raw)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("invalid origin %q: scheme must be http or https", raw)
		}
		// Reconstruct without path/query.
		host := u.Host
		if hasWild {
			host = "*." + strings.TrimPrefix(host, "wildcard.")
		}
		canonical := u.Scheme + "://" + host
		if !seen[canonical] {
			out = append(out, canonical)
			seen[canonical] = true
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// MatchOrigin returns true if origin is permitted by allowed.
// Empty allowed = no restriction (matches anything, used for legacy/non-browser tokens).
// Origin must be a valid http(s) origin string.
func MatchOrigin(allowed []string, origin string) bool {
	if len(allowed) == 0 {
		return true
	}
	if origin == "" {
		return false
	}
	for _, a := range allowed {
		if a == "*" {
			return true
		}
		if a == origin {
			return true
		}
		// Wildcard subdomain match.
		if strings.Contains(a, "://*.") {
			scheme := a[:strings.Index(a, "://")]
			suffix := a[strings.Index(a, "://*.")+len("://*."):]
			prefix := scheme + "://"
			if strings.HasPrefix(origin, prefix) {
				host := strings.TrimPrefix(origin, prefix)
				if host == suffix || strings.HasSuffix(host, "."+suffix) {
					return true
				}
			}
		}
	}
	return false
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
