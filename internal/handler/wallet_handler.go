package handler

import (
	"net/http"

	"github.com/neus/bet-kafka-system/internal/domain"
)

type WalletHandler struct {
	users domain.UserRepository
	txs   domain.TransactionRepository
}

func NewWalletHandler(users domain.UserRepository, txs domain.TransactionRepository) *WalletHandler {
	return &WalletHandler{users: users, txs: txs}
}

func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id required"})
		return
	}

	user, err := h.users.FindByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if user == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	txs, err := h.txs.FindByUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if txs == nil {
		txs = []domain.Transaction{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":      user.ID,
		"username":     user.Username,
		"balance":      user.Balance,
		"transactions": txs,
	})
}
