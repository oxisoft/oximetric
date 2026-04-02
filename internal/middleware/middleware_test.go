package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oxisoft/oximetric/internal/auth"
)

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name     string
		userRole string
		minRole  string
		wantCode int
	}{
		{"admin accessing admin", "admin", "admin", 200},
		{"admin accessing manager", "admin", "manager", 200},
		{"admin accessing viewer", "admin", "viewer", 200},
		{"manager accessing manager", "manager", "manager", 200},
		{"manager accessing viewer", "manager", "viewer", 200},
		{"manager accessing admin", "manager", "admin", 403},
		{"viewer accessing viewer", "viewer", "viewer", 200},
		{"viewer accessing manager", "viewer", "manager", 403},
		{"viewer accessing admin", "viewer", "admin", 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RequireRole(tt.minRole, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			}))

			claims := &auth.Claims{UserID: 1, Username: "test", Role: tt.userRole}
			ctx := context.WithValue(context.Background(), ClaimsKey, claims)

			req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("expected %d, got %d", tt.wantCode, rec.Code)
			}
		})
	}
}

func TestRequireRole_NoClaims(t *testing.T) {
	handler := RequireRole("viewer", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 401 {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestJWTAuth_ValidToken(t *testing.T) {
	svc := auth.NewService("test-secret")
	token, _ := svc.GenerateJWT(1, "admin", "admin")

	handler := JWTAuth(svc, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r.Context())
		if claims == nil {
			t.Error("claims should not be nil")
		}
		if claims.Username != "admin" {
			t.Errorf("expected admin, got %s", claims.Username)
		}
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	svc := auth.NewService("test-secret")
	handler := JWTAuth(svc, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 401 {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	svc := auth.NewService("test-secret")
	handler := JWTAuth(svc, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 401 {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestBodyLimit(t *testing.T) {
	handler := BodyLimit(10, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 100)
		_, err := r.Body.Read(buf)
		if err == nil {
			t.Error("should fail reading beyond limit")
		}
		w.WriteHeader(200)
	}))

	body := make([]byte, 100)
	req := httptest.NewRequest("POST", "/test", bytes(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}

func bytes(b []byte) *bytesReader { return &bytesReader{b: b, i: 0} }

type bytesReader struct {
	b []byte
	i int
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, nil
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}
