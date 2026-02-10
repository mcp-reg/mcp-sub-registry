package handler

import (
	"net/http"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
)

// Health handles GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, model.HealthResponse{Status: "ok"})
}
