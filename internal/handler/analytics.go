package handler

import (
	"net/http"

	"github.com/oxisoft/oximetric/internal/storage"
)

type AnalyticsHandler struct {
	store storage.Store
}

func NewAnalyticsHandler(store storage.Store) *AnalyticsHandler {
	return &AnalyticsHandler{store: store}
}

func (h *AnalyticsHandler) Overview(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathInt(r, "project_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project_id")
		return
	}
	p := parseAnalyticsParams(r)
	result, err := h.store.GetOverview(r.Context(), projectID, p.From, p.To)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get overview")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AnalyticsHandler) Events(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathInt(r, "project_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project_id")
		return
	}
	p := parseAnalyticsParams(r)
	result, err := h.store.GetEventsAnalytics(r.Context(), projectID, p.From, p.To, p.Interval, p.Limit, p.Offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get events analytics")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AnalyticsHandler) EventProperties(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathInt(r, "project_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project_id")
		return
	}
	name := r.PathValue("name")
	p := parseAnalyticsParams(r)
	result, err := h.store.GetEventProperties(r.Context(), projectID, name, p.From, p.To, p.Limit, p.Offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get event properties")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AnalyticsHandler) Devices(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathInt(r, "project_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project_id")
		return
	}
	p := parseAnalyticsParams(r)
	result, err := h.store.GetDevicesAnalytics(r.Context(), projectID, p.From, p.To)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get devices analytics")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AnalyticsHandler) Geo(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathInt(r, "project_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project_id")
		return
	}
	p := parseAnalyticsParams(r)
	result, err := h.store.GetGeoAnalytics(r.Context(), projectID, p.From, p.To, p.Limit, p.Offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get geo analytics")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AnalyticsHandler) Users(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathInt(r, "project_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project_id")
		return
	}
	p := parseAnalyticsParams(r)
	result, err := h.store.GetUsersAnalytics(r.Context(), projectID, p.From, p.To, p.Interval)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get users analytics")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AnalyticsHandler) Retention(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathInt(r, "project_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid project_id")
		return
	}
	p := parseAnalyticsParams(r)
	result, err := h.store.GetRetentionAnalytics(r.Context(), projectID, p.From, p.To)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get retention analytics")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
