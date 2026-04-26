package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/oxisoft/oximetric/internal/auth"
	"github.com/oxisoft/oximetric/internal/middleware"
	"github.com/oxisoft/oximetric/internal/model"
	"github.com/oxisoft/oximetric/internal/storage"
)

type ProjectsHandler struct {
	store  storage.Store
	logger *slog.Logger
}

func NewProjectsHandler(store storage.Store, logger *slog.Logger) *ProjectsHandler {
	return &ProjectsHandler{store: store, logger: logger}
}

func (h *ProjectsHandler) List(w http.ResponseWriter, r *http.Request) {
	projects, err := h.store.ListProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list projects")
		return
	}
	if projects == nil {
		projects = []model.Project{}
	}
	writeJSON(w, http.StatusOK, projects)
}

func (h *ProjectsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}

	project := &model.Project{Name: req.Name}
	if err := h.store.CreateProject(r.Context(), project); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create project")
		return
	}
	claims := middleware.GetClaims(r.Context())
	h.logger.Info("project_created", "by", claims.UserID, "project", project.Name)
	writeJSON(w, http.StatusCreated, project)
}

func (h *ProjectsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project id")
		return
	}

	project, err := h.store.GetProject(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "project not found")
		return
	}

	var req model.UpdateProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.RetentionDays != nil {
		project.RetentionDays = *req.RetentionDays
	}

	if err := h.store.UpdateProject(r.Context(), project); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update project")
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (h *ProjectsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project id")
		return
	}
	if err := h.store.DeleteProject(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete project")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Tokens

func (h *ProjectsHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project id")
		return
	}
	tokens, err := h.store.ListProjectTokens(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list tokens")
		return
	}
	if tokens == nil {
		tokens = []model.ProjectToken{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tokens": tokens})
}

func (h *ProjectsHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project id")
		return
	}

	var req model.CreateTokenRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	tokenStr, err := auth.GenerateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
		return
	}

	origins, err := normalizeOrigins(req.AllowedOrigins)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	token := &model.ProjectToken{
		ProjectID:      id,
		Token:          tokenStr,
		Label:          req.Label,
		Active:         true,
		AllowedOrigins: origins,
	}
	if err := h.store.CreateProjectToken(r.Context(), token); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create token")
		return
	}
	writeJSON(w, http.StatusCreated, token)
}

func (h *ProjectsHandler) UpdateToken(w http.ResponseWriter, r *http.Request) {
	tokenID, err := pathInt(r, "token_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid token id")
		return
	}
	projectID, _ := pathInt(r, "id")

	tokens, err := h.store.ListProjectTokens(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to find token")
		return
	}
	var found *model.ProjectToken
	for i := range tokens {
		if tokens[i].ID == tokenID {
			found = &tokens[i]
			break
		}
	}
	if found == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "token not found")
		return
	}

	var req model.UpdateTokenRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if req.Label != nil {
		found.Label = *req.Label
	}
	if req.AllowedOrigins != nil {
		origins, err := normalizeOrigins(*req.AllowedOrigins)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		found.AllowedOrigins = origins
	}
	if err := h.store.UpdateProjectToken(r.Context(), found); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update token")
		return
	}
	writeJSON(w, http.StatusOK, found)
}

func (h *ProjectsHandler) DisableToken(w http.ResponseWriter, r *http.Request) {
	tokenID, err := pathInt(r, "token_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid token id")
		return
	}

	// Load token list and find by ID
	projectID, _ := pathInt(r, "id")
	tokens, err := h.store.ListProjectTokens(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to find token")
		return
	}

	var found *model.ProjectToken
	for i := range tokens {
		if tokens[i].ID == tokenID {
			found = &tokens[i]
			break
		}
	}
	if found == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "token not found")
		return
	}

	now := time.Now()
	found.Active = false
	found.DisabledAt = &now
	if err := h.store.UpdateProjectToken(r.Context(), found); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to disable token")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *ProjectsHandler) EnableToken(w http.ResponseWriter, r *http.Request) {
	tokenID, err := pathInt(r, "token_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid token id")
		return
	}

	projectID, _ := pathInt(r, "id")
	tokens, err := h.store.ListProjectTokens(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to find token")
		return
	}

	var found *model.ProjectToken
	for i := range tokens {
		if tokens[i].ID == tokenID {
			found = &tokens[i]
			break
		}
	}
	if found == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "token not found")
		return
	}

	found.Active = true
	found.DisabledAt = nil
	if err := h.store.UpdateProjectToken(r.Context(), found); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to enable token")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *ProjectsHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	tokenID, err := pathInt(r, "token_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid token id")
		return
	}
	if err := h.store.DeleteProjectToken(r.Context(), tokenID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete token")
		return
	}
	w.WriteHeader(http.StatusOK)
}
