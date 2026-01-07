package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"tell/internal/auth"

	"gorm.io/gorm"
)

type AuthHandler struct {
	DB  *gorm.DB
	JWT *auth.JWT
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || len(req.Password) < 8 {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	u := auth.User{Email: req.Email, PasswordHash: hash}
	if err := h.DB.Create(&u).Error; err != nil {
		http.Error(w, "email already used", http.StatusConflict)
		return
	}

	token, err := h.JWT.Sign(u.ID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token": token,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}

	var u auth.User
	if err := h.DB.Where("email = ?", req.Email).First(&u).Error; err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if !auth.ComparePassword(u.PasswordHash, req.Password) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := h.JWT.Sign(u.ID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token": token,
	})
}
