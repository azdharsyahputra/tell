package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tell/internal/auth"
	"tell/internal/jobs"
	"tell/internal/memo"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type MemoHandler struct {
	Svc *memo.Service
	DB  *gorm.DB
}

type createMemoReq struct {
	Content  string  `json:"content"`
	RemindAt *string `json:"remind_at"` // RFC3339 optional
}

func (h *MemoHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())

	var req createMemoReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		http.Error(w, "content required", http.StatusBadRequest)
		return
	}

	var remindAt *time.Time
	if req.RemindAt != nil && strings.TrimSpace(*req.RemindAt) != "" {
		t, err := time.Parse(time.RFC3339, *req.RemindAt)
		if err != nil {
			http.Error(w, "invalid remind_at (RFC3339)", http.StatusBadRequest)
			return
		}
		remindAt = &t
	}

	var idem *string
	if k := strings.TrimSpace(r.Header.Get("Idempotency-Key")); k != "" {
		idem = &k
	}

	id, err := h.Svc.CreateMemo(r.Context(), uid, memo.CreateMemoInput{
		Content:  req.Content,
		RemindAt: remindAt,
		IdemKey:  idem,
	})
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"id": id})
}

type appendEventReq struct {
	Type     string  `json:"type"`
	Content  *string `json:"content"`
	RemindAt *string `json:"remind_at"`
}

func (h *MemoHandler) AppendEvent(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req appendEventReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	req.Type = strings.TrimSpace(strings.ToUpper(req.Type))

	var remindAt *time.Time
	if req.RemindAt != nil && strings.TrimSpace(*req.RemindAt) != "" {
		t, err := time.Parse(time.RFC3339, *req.RemindAt)
		if err != nil {
			http.Error(w, "invalid remind_at (RFC3339)", http.StatusBadRequest)
			return
		}
		remindAt = &t
	}

	var idem *string
	if k := strings.TrimSpace(r.Header.Get("Idempotency-Key")); k != "" {
		idem = &k
	}

	// 1) apply domain event first
	err = h.Svc.AppendEvent(r.Context(), memo.AppendEventInput{
		MemoID:   id64,
		UserID:   uid,
		Type:     req.Type,
		Content:  req.Content,
		RemindAt: remindAt,
		IdemKey:  idem,
	})

	// 2) handle errors BEFORE any job side-effect
	if err != nil {
		switch err {
		case memo.ErrNotFound:
			http.Error(w, "not found", http.StatusNotFound)
			return
		case memo.ErrInvalidEvent:
			http.Error(w, "invalid event", http.StatusBadRequest)
			return
		default:
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	}

	// 3) job side-effects (DB required)
	if (req.Type == "REMINDER_SET" && remindAt != nil) || req.Type == "REMINDER_CLEARED" {
		if h.DB == nil {
			http.Error(w, "server misconfigured (db)", http.StatusInternalServerError)
			return
		}
	}

	// 3a) REMINDER_SET: dedupe pending reminder jobs for this memo, then enqueue
	if req.Type == "REMINDER_SET" && remindAt != nil {
		// delete any pending reminder job for this memo (avoid double dispatch)
		_ = h.DB.Exec(`
			delete from jobs
			where user_id = ?
			  and type = 'REMINDER_DISPATCH'
			  and status = 'PENDING'
			  and (payload->>'memo_id')::bigint = ?
		`, uid, id64).Error

		payload, _ := json.Marshal(map[string]any{"memo_id": id64})
		j := jobs.Job{
			UserID:  uid,
			Type:    "REMINDER_DISPATCH",
			Payload: payload,
			RunAt:   *remindAt,
			Status:  "PENDING",
		}
		if err := h.DB.Create(&j).Error; err != nil {
			http.Error(w, "failed enqueue job", http.StatusInternalServerError)
			return
		}
	}

	// 3b) REMINDER_CLEARED: cancel pending reminder jobs
	if req.Type == "REMINDER_CLEARED" {
		if err := h.DB.Exec(`
			delete from jobs
			where user_id = ?
			  and type = 'REMINDER_DISPATCH'
			  and status = 'PENDING'
			  and (payload->>'memo_id')::bigint = ?
		`, uid, id64).Error; err != nil {
			http.Error(w, "failed cancel reminder jobs", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
