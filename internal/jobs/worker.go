package jobs

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"time"

	"gorm.io/gorm"
)

type Worker struct {
	ID   string
	Repo *Repo
	DB   *gorm.DB
}

type memoProjection struct {
	MemoID   uint64     `gorm:"column:memo_id"`
	UserID   uint64     `gorm:"column:user_id"`
	Content  string     `gorm:"column:content"`
	Archived bool       `gorm:"column:archived"`
	RemindAt *time.Time `gorm:"column:remind_at"`
}

func (memoProjection) TableName() string { return "memo_projections" }

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(800 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			job, err := w.Repo.Claim(w.ID)
			if err != nil {
				log.Printf("worker claim error: %v\n", err)
				continue
			}
			if job == nil {
				continue
			}
			w.handle(job)
		}
	}
}

func (w *Worker) handle(job *Job) {
	switch job.Type {
	case "REMINDER_DISPATCH":
		w.handleReminder(job)
	default:
		_ = w.Repo.MarkFailed(job.ID, "unknown job type")
	}
}

func (w *Worker) handleReminder(job *Job) {
	type payload struct {
		MemoID uint64 `json:"memo_id"`
	}
	var p payload
	if err := json.Unmarshal(job.Payload, &p); err != nil {
		_ = w.Repo.MarkFailed(job.ID, "bad payload")
		return
	}

	var proj memoProjection
	if err := w.DB.
		Where("memo_id=? AND user_id=?", p.MemoID, job.UserID).
		First(&proj).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			_ = w.Repo.MarkDone(job.ID)
			return
		}
		w.retry(job, "db read error")
		return
	}

	if proj.Archived || proj.RemindAt == nil {
		_ = w.Repo.MarkDone(job.ID)
		return
	}

	log.Printf("[REMINDER] user=%d memo=%d content=%q\n", job.UserID, proj.MemoID, proj.Content)
	_ = w.Repo.MarkDone(job.ID)
}

func (w *Worker) retry(job *Job, errMsg string) {
	attempts := job.Attempts + 1
	if attempts >= job.MaxAttempts {
		_ = w.Repo.MarkFailed(job.ID, errMsg)
		return
	}

	sec := math.Min(math.Pow(2, float64(attempts)), 600)
	next := time.Now().Add(time.Duration(sec) * time.Second)

	_ = w.Repo.RetryLater(job.ID, attempts, next, errMsg)
}
