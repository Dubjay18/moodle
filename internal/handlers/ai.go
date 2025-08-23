package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/yourname/moodle/internal/ai"
	"github.com/yourname/moodle/internal/validate"
)

type AIHandler struct{ AI *ai.GeminiClient }

func NewAIHandler(c *ai.GeminiClient) *AIHandler { return &AIHandler{AI: c} }

func (h *AIHandler) Ask(w http.ResponseWriter, r *http.Request) {
	type req struct {
		Query string `json:"query" validate:"required,min=1,max=500"`
	}
	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}
	if errs := validate.Map(body); errs != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(errs)
		return
	}
	answer, err := h.AI.Ask(r.Context(), body.Query)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"answer": answer})
}
