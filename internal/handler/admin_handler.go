package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/neus/bet-kafka-system/internal/domain"
)

type AdminHandler struct {
	events domain.EventRepository
	pub    domain.EventPublisher
}

func NewAdminHandler(events domain.EventRepository, pub domain.EventPublisher) *AdminHandler {
	return &AdminHandler{events: events, pub: pub}
}

type SettleRequest struct {
	EventID          string `json:"event_id"`
	MarketID         string `json:"market_id"`
	WinningOutcomeID string `json:"winning_outcome_id"`
}

func (h *AdminHandler) SettleMarket(w http.ResponseWriter, r *http.Request) {
	var req SettleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.EventID == "" || req.MarketID == "" || req.WinningOutcomeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required fields"})
		return
	}

	correlationID := uuid.New().String()
	if err := h.pub.PublishBetSettled(r.Context(), req.EventID, req.MarketID, req.WinningOutcomeID, correlationID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to settle"})
		return
	}

	// Update market status in MongoDB
	_ = h.events.UpdateMarketStatus(r.Context(), req.EventID, req.MarketID, domain.MarketStatusSettled)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"correlation_id": correlationID,
		"status":         "SETTLEMENT_QUEUED",
	})
}

type EventStatusRequest struct {
	EventID string              `json:"event_id"`
	Status  domain.EventStatus  `json:"status"`
}

func (h *AdminHandler) UpdateEventStatus(w http.ResponseWriter, r *http.Request) {
	var req EventStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.events.UpdateStatus(r.Context(), req.EventID, req.Status); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"event_id": req.EventID,
		"status":   req.Status,
	})
}
