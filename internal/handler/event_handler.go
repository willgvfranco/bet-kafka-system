package handler

import (
	"net/http"

	"github.com/neus/bet-kafka-system/internal/domain"
)

type EventHandler struct {
	events domain.EventRepository
	odds   domain.OddsCache
}

func NewEventHandler(events domain.EventRepository, odds domain.OddsCache) *EventHandler {
	return &EventHandler{events: events, odds: odds}
}

func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.events.FindAll(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if events == nil {
		events = []domain.Event{}
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	event, err := h.events.FindByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if event == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
		return
	}

	// Enrich with live odds from Redis
	liveOdds, _ := h.odds.GetEventOdds(r.Context(), event.ID)
	type enrichedOutcome struct {
		domain.Outcome
		LiveOdds float64 `json:"live_odds,omitempty"`
	}
	type enrichedMarket struct {
		MarketID string                `json:"market_id"`
		Type     domain.MarketType     `json:"type"`
		Status   domain.MarketStatus   `json:"status"`
		Outcomes []enrichedOutcome     `json:"outcomes"`
	}
	type enrichedEvent struct {
		domain.Event
		Markets []enrichedMarket `json:"markets"`
	}

	result := enrichedEvent{Event: *event}
	result.Markets = make([]enrichedMarket, len(event.Markets))
	for i, m := range event.Markets {
		result.Markets[i] = enrichedMarket{
			MarketID: m.MarketID,
			Type:     m.Type,
			Status:   m.Status,
			Outcomes: make([]enrichedOutcome, len(m.Outcomes)),
		}
		for j, o := range m.Outcomes {
			eo := enrichedOutcome{Outcome: o}
			if odds, ok := liveOdds[o.OutcomeID]; ok {
				eo.LiveOdds = odds
			}
			result.Markets[i].Outcomes[j] = eo
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *EventHandler) GetEventOdds(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	odds, err := h.odds.GetEventOdds(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if odds == nil {
		odds = map[string]float64{}
	}
	writeJSON(w, http.StatusOK, odds)
}
