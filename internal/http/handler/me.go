package handler

import (
	"encoding/json"
	"net/http"

	"tell/internal/auth"
)

type MeHandler struct{}

func (h *MeHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user_id": uid,
	})
}
