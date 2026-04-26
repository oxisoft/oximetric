package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/oxisoft/oximetric/internal/auth"
	"github.com/oxisoft/oximetric/internal/model"
	"github.com/oxisoft/oximetric/internal/storage"
)

const corsMaxAge = "86400" // 24h
const corsAllowedHeaders = "X-Token, Content-Type"
const corsAllowedMethods = "POST, OPTIONS"

type contextKey string

const (
	ProjectIDKey contextKey = "project_id"
	ClaimsKey    contextKey = "claims"
)

func Logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration", time.Since(start).String(),
		)
	})
}

func BodyLimit(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next.ServeHTTP(w, r)
	})
}

func TokenAuth(store storage.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Token")
		if token == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing X-Token header")
			return
		}
		pt, err := store.GetProjectTokenByToken(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
			return
		}
		if !pt.Active {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "token is disabled")
			return
		}
		// Enforce per-token allowed origins. Empty list = no restriction (server-side / mobile SDK).
		origin := r.Header.Get("Origin")
		if len(pt.AllowedOrigins) > 0 {
			if !matchOrigin(pt.AllowedOrigins, origin) {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "origin not allowed for this token")
				return
			}
			// Echo the matched origin so the browser accepts the response.
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		ctx := context.WithValue(r.Context(), ProjectIDKey, pt.ProjectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CORS handles preflight (OPTIONS) requests for tracking endpoints.
// It responds with permissive headers because the actual security check
// happens in TokenAuth on the real request (which validates Origin against
// the token's allowed list). Without this middleware, browsers cannot
// preflight cross-origin POSTs.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if r.Method == http.MethodOptions && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", corsAllowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
			w.Header().Set("Access-Control-Max-Age", corsMaxAge)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// matchOrigin returns true if origin is in allowed.
// Supports exact match, the literal "*", and "scheme://*.host" wildcards.
func matchOrigin(allowed []string, origin string) bool {
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

func JWTAuth(svc *auth.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid Authorization header")
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := svc.ValidateJWT(tokenStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(minRole string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r.Context())
		if claims == nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
			return
		}
		if !hasMinRole(claims.Role, minRole) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RateLimit(requestsPerMinute int, next http.Handler) http.Handler {
	type bucket struct {
		tokens int
		last   time.Time
	}
	var mu sync.Mutex
	buckets := make(map[string]*bucket)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Token")
		if key == "" {
			key = r.RemoteAddr
		}

		mu.Lock()
		b, ok := buckets[key]
		if !ok {
			b = &bucket{tokens: requestsPerMinute, last: time.Now()}
			buckets[key] = b
		}
		elapsed := time.Since(b.last)
		b.tokens += int(elapsed.Minutes() * float64(requestsPerMinute))
		if b.tokens > requestsPerMinute {
			b.tokens = requestsPerMinute
		}
		b.last = time.Now()

		if b.tokens <= 0 {
			mu.Unlock()
			writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "rate limit exceeded")
			return
		}
		b.tokens--
		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// Helpers

func GetProjectID(ctx context.Context) int {
	v, _ := ctx.Value(ProjectIDKey).(int)
	return v
}

func GetClaims(ctx context.Context) *auth.Claims {
	v, _ := ctx.Value(ClaimsKey).(*auth.Claims)
	return v
}

func hasMinRole(userRole, minRole string) bool {
	roles := map[string]int{"viewer": 1, "manager": 2, "admin": 3}
	return roles[userRole] >= roles[minRole]
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(model.ErrorResponse{
		Error: model.ErrorDetail{Code: code, Message: message},
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
