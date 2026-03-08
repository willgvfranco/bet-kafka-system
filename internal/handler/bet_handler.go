package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/neus/bet-kafka-system/internal/domain"
)

type BetHandler struct {
	bets domain.BetRepository
	pub  domain.EventPublisher
}

func NewBetHandler(bets domain.BetRepository, pub domain.EventPublisher) *BetHandler {
	return &BetHandler{bets: bets, pub: pub}
}

type PlaceBetRequest struct {
	UserID    string  `json:"user_id"`
	EventID   string  `json:"event_id"`
	MarketID  string  `json:"market_id"`
	OutcomeID string  `json:"outcome_id"`
	Stake     float64 `json:"stake"`
	Odds      float64 `json:"odds"`
}

func (h *BetHandler) PlaceBet(w http.ResponseWriter, r *http.Request) {
	var req PlaceBetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.UserID == "" || req.EventID == "" || req.MarketID == "" || req.OutcomeID == "" || req.Stake <= 0 || req.Odds <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required fields"})
		return
	}

	bet := &domain.Bet{
		ID:              uuid.New().String(),
		UserID:          req.UserID,
		EventID:         req.EventID,
		MarketID:        req.MarketID,
		OutcomeID:       req.OutcomeID,
		Stake:           req.Stake,
		OddsAtPlacement: req.Odds,
		PotentialPayout: req.Stake * req.Odds,
		IdempotencyKey:  uuid.New().String(),
		CorrelationID:   uuid.New().String(),
		PlacedAt:        time.Now(),
	}

	if err := h.pub.PublishBetPlaced(r.Context(), bet); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to place bet"})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"bet_id":         bet.ID,
		"correlation_id": bet.CorrelationID,
		"status":         "ACCEPTED",
	})
}

func (h *BetHandler) GetBets(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id required"})
		return
	}
	bets, err := h.bets.FindByUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if bets == nil {
		bets = []domain.Bet{}
	}
	writeJSON(w, http.StatusOK, bets)
}

func (h *BetHandler) GetBetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	bet, err := h.bets.FindByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if bet == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "bet not found"})
		return
	}
	writeJSON(w, http.StatusOK, bet)
}
