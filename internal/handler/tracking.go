package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/oxisoft/oximetric/internal/geoip"
	"github.com/oxisoft/oximetric/internal/middleware"
	"github.com/oxisoft/oximetric/internal/model"
	"github.com/oxisoft/oximetric/internal/storage"
)

const (
	maxEventsPerBatch    = 100
	maxPropertiesPerEvent = 50
	maxEventNameLen      = 256
	maxKeyLen            = 128
	maxStringValueLen    = 4096
)

type TrackingHandler struct {
	store          storage.Store
	geo            *geoip.Resolver
	trustedProxies map[string]bool
}

func NewTrackingHandler(store storage.Store, geo *geoip.Resolver, trustedProxies []string) *TrackingHandler {
	tp := make(map[string]bool)
	for _, p := range trustedProxies {
		tp[p] = true
	}
	return &TrackingHandler{store: store, geo: geo, trustedProxies: tp}
}

func (h *TrackingHandler) Track(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.GetProjectID(r.Context())

	var req model.TrackRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "device_id is required")
		return
	}
	if len(req.Events) == 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "events array is empty")
		return
	}
	if len(req.Events) > maxEventsPerBatch {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", fmt.Sprintf("max %d events per batch", maxEventsPerBatch))
		return
	}

	// Resolve IP
	ip := h.clientIPTrusted(r)
	var country, city string
	if h.geo != nil {
		loc := h.geo.Lookup(ip)
		country = loc.Country
		city = loc.City
	}

	// Fallback user_id from device if not provided
	userID := req.UserID
	if userID == nil {
		userID, _ = h.store.GetDeviceUserID(r.Context(), req.DeviceID)
	}

	accepted := 0
	for _, ev := range req.Events {
		if ev.ID == "" || ev.Name == "" {
			continue
		}
		if len(ev.Name) > maxEventNameLen {
			continue
		}
		if len(ev.Properties) > maxPropertiesPerEvent {
			continue
		}

		event := &model.Event{
			ID:        ev.ID,
			ProjectID: projectID,
			DeviceID:  req.DeviceID,
			UserID:    userID,
			Name:      ev.Name,
			Timestamp: ev.Timestamp,
			IPAddress: ip,
			Country:   country,
			City:      city,
		}

		inserted, err := h.store.InsertEvent(r.Context(), event)
		if err != nil || !inserted {
			continue
		}

		if len(ev.Properties) > 0 {
			props := buildProperties(ev.ID, ev.Properties)
			h.store.InsertEventProperties(r.Context(), props)
		}
		accepted++
	}

	writeJSON(w, http.StatusAccepted, model.TrackResponse{Accepted: accepted})
}

func (h *TrackingHandler) Device(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.GetProjectID(r.Context())

	var req model.DeviceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if req.DeviceID == "" || req.Platform == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "device_id and platform are required")
		return
	}

	device := &model.Device{
		ID:         req.DeviceID,
		ProjectID:  projectID,
		Platform:   req.Platform,
		OSVersion:  req.OSVersion,
		AppVersion: req.AppVersion,
		Locale:     req.Locale,
	}
	if err := h.store.UpsertDevice(r.Context(), device); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to register device")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TrackingHandler) Identify(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.GetProjectID(r.Context())

	var req model.IdentifyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if req.DeviceID == "" || req.AnonymousID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "device_id and anonymous_id are required")
		return
	}

	user, err := h.store.GetOrCreateAppUser(r.Context(), projectID, req.AnonymousID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create user")
		return
	}

	if err := h.store.LinkDeviceToUser(r.Context(), req.DeviceID, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to link device")
		return
	}

	writeJSON(w, http.StatusOK, model.IdentifyResponse{UserID: user.ID})
}

func buildProperties(eventID string, props map[string]model.TrackPropertyValue) []model.EventProperty {
	var result []model.EventProperty
	for key, pv := range props {
		if len(key) > maxKeyLen {
			continue
		}
		ep := model.EventProperty{
			EventID:   eventID,
			Key:       key,
			ValueType: pv.Type,
		}
		switch pv.Type {
		case "string":
			if s, ok := pv.Value.(string); ok {
				if len(s) > maxStringValueLen {
					s = s[:maxStringValueLen]
				}
				ep.ValueString = &s
			}
		case "int":
			if f, ok := pv.Value.(float64); ok {
				v := int64(f)
				ep.ValueInt = &v
			}
		case "float":
			if f, ok := pv.Value.(float64); ok {
				ep.ValueFloat = &f
			}
		case "bool":
			if b, ok := pv.Value.(bool); ok {
				ep.ValueBool = &b
			}
		case "datetime":
			if s, ok := pv.Value.(string); ok {
				if t, err := time.Parse(time.RFC3339, s); err == nil {
					ep.ValueDatetime = &t
				}
			}
		}
		result = append(result, ep)
	}
	return result
}

func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host
}

func (h *TrackingHandler) clientIPTrusted(r *http.Request) string {
	remoteIP := clientIP(r)
	// If no trusted proxies configured, trust forwarded headers from anyone
	// If configured, only trust from listed IPs
	trusted := len(h.trustedProxies) == 0 || h.trustedProxies[remoteIP]
	if trusted {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.SplitN(xff, ",", 2)
			return strings.TrimSpace(parts[0])
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}
	}
	return remoteIP
}
