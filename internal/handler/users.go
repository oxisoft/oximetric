package handler

import (
	"log/slog"
	"net/http"

	"github.com/oxisoft/oximetric/internal/auth"
	"github.com/oxisoft/oximetric/internal/middleware"
	"github.com/oxisoft/oximetric/internal/model"
	"github.com/oxisoft/oximetric/internal/storage"
)

type UsersHandler struct {
	store  storage.Store
	logger *slog.Logger
}

func NewUsersHandler(store storage.Store, logger *slog.Logger) *UsersHandler {
	return &UsersHandler{store: store, logger: logger}
}

func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.ListConsoleUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list users")
		return
	}
	if users == nil {
		users = []model.ConsoleUser{}
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *UsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	var req model.CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" || req.Role == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "username, password, and role are required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "password must be at least 8 characters")
		return
	}
	if req.Role != "admin" && req.Role != "manager" && req.Role != "viewer" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "role must be admin, manager, or viewer")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
		return
	}

	user := &model.ConsoleUser{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         req.Role,
	}
	if err := h.store.CreateConsoleUser(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create user")
		return
	}
	h.logger.Info("user_created", "created_by", claims.UserID, "username", req.Username, "role", req.Role)
	writeJSON(w, http.StatusCreated, user)
}

func (h *UsersHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user id")
		return
	}
	if id == claims.UserID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "cannot modify your own account here, use Account settings")
		return
	}

	user, err := h.store.GetConsoleUserByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	var req model.UpdateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.Email != nil {
		user.Email = req.Email
	}
	if req.Role != nil {
		if *req.Role != "admin" && *req.Role != "manager" && *req.Role != "viewer" {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "role must be admin, manager, or viewer")
			return
		}
		user.Role = *req.Role
	}

	if err := h.store.UpdateConsoleUser(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update user")
		return
	}
	h.logger.Info("user_updated", "updated_by", claims.UserID, "target_id", id)
	writeJSON(w, http.StatusOK, user)
}

func (h *UsersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user id")
		return
	}
	if id == claims.UserID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "cannot delete your own account")
		return
	}
	if err := h.store.DeleteConsoleUser(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete user")
		return
	}
	h.logger.Info("user_deleted", "deleted_by", claims.UserID, "target_id", id)
	w.WriteHeader(http.StatusOK)
}
