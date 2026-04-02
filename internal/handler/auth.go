package handler

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/oxisoft/oximetric/internal/auth"
	"github.com/oxisoft/oximetric/internal/middleware"
	"github.com/oxisoft/oximetric/internal/model"
	"github.com/oxisoft/oximetric/internal/storage"
)

type AuthHandler struct {
	store   storage.Store
	authSvc *auth.Service
	logger  *slog.Logger

	// TOTP brute force protection
	totpAttempts   map[string]*totpTracker
	totpAttemptsMu sync.Mutex
}

type totpTracker struct {
	failures int
	resetAt  time.Time
}

func NewAuthHandler(store storage.Store, authSvc *auth.Service, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		store:        store,
		authSvc:      authSvc,
		logger:       logger,
		totpAttempts: make(map[string]*totpTracker),
	}
}

func (h *AuthHandler) checkTOTPLimit(key string) bool {
	h.totpAttemptsMu.Lock()
	defer h.totpAttemptsMu.Unlock()
	t, ok := h.totpAttempts[key]
	if !ok || time.Now().After(t.resetAt) {
		return true // allowed
	}
	return t.failures < 5
}

func (h *AuthHandler) recordTOTPFailure(key string) {
	h.totpAttemptsMu.Lock()
	defer h.totpAttemptsMu.Unlock()
	t, ok := h.totpAttempts[key]
	if !ok || time.Now().After(t.resetAt) {
		h.totpAttempts[key] = &totpTracker{failures: 1, resetAt: time.Now().Add(5 * time.Minute)}
		return
	}
	t.failures++
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if req.Login == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "login and password are required")
		return
	}

	user, err := h.store.GetConsoleUserByLogin(r.Context(), req.Login)
	if err != nil {
		// Timing attack prevention: always run bcrypt even if user not found
		auth.CheckPassword(string(auth.DummyHash), req.Password)
		h.logger.Info("login_failed", "login", req.Login, "reason", "user_not_found", "ip", clientIP(r))
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		h.logger.Info("login_failed", "login", req.Login, "reason", "bad_password", "ip", clientIP(r))
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
		return
	}

	if user.TOTPEnabled && user.TOTPSecret != nil {
		if req.TOTPCode == "" {
			writeJSON(w, http.StatusOK, model.LoginResponse{TOTPRequired: true})
			return
		}
		totpKey := req.Login
		if !h.checkTOTPLimit(totpKey) {
			h.logger.Warn("totp_rate_limited", "login", req.Login, "ip", clientIP(r))
			writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many TOTP attempts, try again later")
			return
		}
		if !auth.ValidateTOTPCode(*user.TOTPSecret, req.TOTPCode) {
			h.recordTOTPFailure(totpKey)
			h.logger.Info("login_failed", "login", req.Login, "reason", "bad_totp", "ip", clientIP(r))
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid TOTP code")
			return
		}
	}

	token, err := h.authSvc.GenerateJWT(user.ID, user.Username, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
		return
	}

	h.logger.Info("login_success", "username", user.Username, "ip", clientIP(r))
	writeJSON(w, http.StatusOK, model.LoginResponse{Token: token, User: user})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	user, err := h.store.GetConsoleUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())

	var req model.PasswordChangeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "password must be at least 8 characters")
		return
	}

	user, err := h.store.GetConsoleUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, req.CurrentPassword) {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "current password is incorrect")
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
		return
	}
	user.PasswordHash = hash
	if err := h.store.UpdateConsoleUser(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update password")
		return
	}

	h.logger.Info("password_changed", "user_id", claims.UserID, "ip", clientIP(r))
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) TOTPSetup(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())

	secret, uri, err := auth.GenerateTOTPSecret(claims.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate TOTP secret")
		return
	}

	user, err := h.store.GetConsoleUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	user.TOTPSecret = &secret
	if err := h.store.UpdateConsoleUser(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to save TOTP secret")
		return
	}

	writeJSON(w, http.StatusOK, model.TOTPSetupResponse{Secret: secret, URI: uri})
}

func (h *AuthHandler) TOTPEnable(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())

	var req model.TOTPEnableRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	user, err := h.store.GetConsoleUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	// Require password to enable 2FA
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "incorrect password")
		return
	}

	if user.TOTPSecret == nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "call /auth/totp/setup first")
		return
	}
	if !auth.ValidateTOTPCode(*user.TOTPSecret, req.Code) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid TOTP code")
		return
	}

	user.TOTPEnabled = true
	if err := h.store.UpdateConsoleUser(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to enable TOTP")
		return
	}

	h.logger.Info("totp_enabled", "user_id", claims.UserID, "ip", clientIP(r))
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) TOTPDisable(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())

	var req model.TOTPDisableRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	user, err := h.store.GetConsoleUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "incorrect password")
		return
	}

	user.TOTPEnabled = false
	user.TOTPSecret = nil
	if err := h.store.UpdateConsoleUser(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to disable TOTP")
		return
	}

	h.logger.Info("totp_disabled", "user_id", claims.UserID, "ip", clientIP(r))
	w.WriteHeader(http.StatusOK)
}
