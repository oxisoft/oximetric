package handler

import (
	"net/http"

	"github.com/oxisoft/oximetric/internal/geoip"
	"github.com/oxisoft/oximetric/internal/model"
	"github.com/oxisoft/oximetric/internal/storage"
)

type HealthHandler struct {
	store   storage.Store
	geoip   *geoip.Resolver
	version string
}

func NewHealthHandler(store storage.Store, geo *geoip.Resolver, version string) *HealthHandler {
	return &HealthHandler{store: store, geoip: geo, version: version}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	resp := model.HealthResponse{
		Status:   "ok",
		Database: "ok",
		GeoIP:    "ok",
		Version:  h.version,
	}

	if err := h.store.Ping(r.Context()); err != nil {
		resp.Status = "degraded"
		resp.Database = "error"
	}
	if h.geoip == nil {
		resp.Status = "degraded"
		resp.GeoIP = "unavailable"
	}

	status := http.StatusOK
	if resp.Status != "ok" {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, resp)
}
