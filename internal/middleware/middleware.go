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
		ctx := context.WithValue(r.Context(), ProjectIDKey, pt.ProjectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
