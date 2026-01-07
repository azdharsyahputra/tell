package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tell/internal/auth"
	"tell/internal/memo"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type MemoReadHandler struct {
	DB *gorm.DB
}

type memoDTO struct {
	MemoID    uint64     `json:"memo_id"`
	UserID    uint64     `json:"user_id"`
	Content   string     `json:"content"`
	Archived  bool       `json:"archived"`
	RemindAt  *time.Time `json:"remind_at"`
	Tags      []string   `json:"tags"`
	Version   uint64     `json:"version"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type memoEventDTO struct {
	ID             uint64          `json:"id"`
	MemoID         uint64          `json:"memo_id"`
	UserID         uint64          `json:"user_id"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	IdempotencyKey *string         `json:"idempotency_key"`
	CreatedAt      time.Time       `json:"created_at"`
}

func (h *MemoReadHandler) List(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())

	tag := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("tag")))
	archived := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("archived"))) // "true"/"false"/""

	// ✅ NEW: search query
	qText := strings.TrimSpace(r.URL.Query().Get("q"))

	q := h.DB.Model(&memo.MemoProjection{}).Where("user_id = ?", uid)

	if archived == "true" {
		q = q.Where("archived = true")
	} else if archived == "false" {
		q = q.Where("archived = false")
	}

	if tag != "" {
		q = q.Where("? = any(tags)", tag)
	}

	// ✅ NEW: search by content (case-insensitive)
	if qText != "" {
		q = q.Where("content ILIKE ?", "%"+qText+"%")
	}

	var rows []memo.MemoProjection
	if err := q.Order("updated_at desc").Limit(50).Find(&rows).Error; err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	out := make([]memoDTO, 0, len(rows))
	for _, p := range rows {
		out = append(out, memoDTO{
			MemoID:    p.MemoID,
			UserID:    p.UserID,
			Content:   p.Content,
			Archived:  p.Archived,
			RemindAt:  p.RemindAt,
			Tags:      []string(p.Tags),
			Version:   p.Version,
			UpdatedAt: p.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *MemoReadHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// verify ownership
	var m memo.Memo
	if err := h.DB.Where("id=? AND user_id=?", id64, uid).First(&m).Error; err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var evs []memo.MemoEvent
	if err := h.DB.Where("memo_id=? AND user_id=?", id64, uid).Order("id asc").Find(&evs).Error; err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	out := make([]memoEventDTO, 0, len(evs))
	for _, e := range evs {
		out = append(out, memoEventDTO{
			ID:             e.ID,
			MemoID:         e.MemoID,
			UserID:         e.UserID,
			Type:           e.Type,
			Payload:        e.Payload,
			IdempotencyKey: e.IdempotencyKey,
			CreatedAt:      e.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

type tagDTO struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

func (h *MemoReadHandler) Tags(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())

	qText := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("q")))

	limit := 50
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	var out []tagDTO
	if err := h.DB.Raw(`
		select tag, count(*) as count
		from (
			select unnest(tags) as tag
			from memo_projections
			where user_id = ? and archived = false
		) t
		where (? = '' or tag like ? || '%')
		group by tag
		order by count desc, tag asc
		limit ?
	`, uid, qText, qText, limit).Scan(&out).Error; err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
