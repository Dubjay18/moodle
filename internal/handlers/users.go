package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/yourname/moodle/internal/auth"
	"github.com/yourname/moodle/internal/store"
)

type UserHandler struct{ Store *store.Store }

func NewUserHandler(s *store.Store) *UserHandler { return &UserHandler{Store: s} }

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid := auth.UserID(r.Context())
	if uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	u, err := h.Store.GetUser(r.Context(), uid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user not found"})
		return
	}
	_ = json.NewEncoder(w).Encode(u)
}
