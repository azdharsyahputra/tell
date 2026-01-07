package memo

import (
	"context"
	"encoding/json"
	"errors"
	"tell/internal/jobs"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("not found")
var ErrInvalidEvent = errors.New("invalid event")

type Service struct {
	DB *gorm.DB
}

type CreateMemoInput struct {
	Content  string
	RemindAt *time.Time
	IdemKey  *string
}

type AppendEventInput struct {
	MemoID   uint64
	UserID   uint64
	Type     string
	Content  *string
	RemindAt *time.Time
	IdemKey  *string
}

func (s *Service) CreateMemo(ctx context.Context, userID uint64, in CreateMemoInput) (uint64, error) {
	var memoID uint64

	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		m := Memo{UserID: userID}
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		memoID = m.ID

		// CREATED event
		if err := s.insertEvent(tx, memoID, userID, "CREATED", map[string]any{
			"content": in.Content,
		}, in.IdemKey); err != nil {
			return err
		}

		// Projection
		proj := MemoProjection{
			MemoID:    memoID,
			UserID:    userID,
			Content:   in.Content,
			Archived:  false,
			RemindAt:  nil,
			Tags:      pq.StringArray(ExtractTags(in.Content)),
			Version:   0,
			UpdatedAt: time.Now(),
		}
		if err := tx.Create(&proj).Error; err != nil {
			return err
		}

		// If remind_at provided: add event + update projection + enqueue job (atomic)
		if in.RemindAt != nil {
			if err := s.insertEvent(tx, memoID, userID, "REMINDER_SET", map[string]any{
				"remind_at": in.RemindAt.Format(time.RFC3339),
			}, nil); err != nil {
				return err
			}

			// enqueue job using SAME tx
			payload, _ := json.Marshal(map[string]any{"memo_id": memoID})
			j := jobs.Job{
				UserID:  userID,
				Type:    "REMINDER_DISPATCH",
				Payload: payload,
				RunAt:   *in.RemindAt,
				Status:  "PENDING",
			}
			if err := tx.Create(&j).Error; err != nil {
				return err
			}

			proj.RemindAt = in.RemindAt
		}

		// set version to last event id (once at the end)
		var last MemoEvent
		if err := tx.Where("memo_id=? AND user_id=?", memoID, userID).Order("id desc").First(&last).Error; err != nil {
			return err
		}

		return tx.Model(&MemoProjection{}).
			Where("memo_id=? AND user_id=?", memoID, userID).
			Updates(map[string]any{
				"remind_at":  proj.RemindAt,
				"version":    last.ID,
				"updated_at": time.Now(),
			}).Error
	})

	return memoID, err
}

func (s *Service) AppendEvent(ctx context.Context, in AppendEventInput) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// ensure memo belongs to user
		var m Memo
		if err := tx.Where("id=? AND user_id=?", in.MemoID, in.UserID).First(&m).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		payload := map[string]any{}
		switch in.Type {
		case "UPDATED":
			if in.Content == nil {
				return ErrInvalidEvent
			}
			payload["content"] = *in.Content
		case "ARCHIVED", "RESTORED":
		case "REMINDER_SET":
			if in.RemindAt == nil {
				return ErrInvalidEvent
			}
			payload["remind_at"] = in.RemindAt.Format(time.RFC3339)
		case "REMINDER_CLEARED":
		default:
			return ErrInvalidEvent
		}

		if err := s.insertEvent(tx, in.MemoID, in.UserID, in.Type, payload, in.IdemKey); err != nil {
			return err
		}

		// fetch projection FOR UPDATE
		var p MemoProjection
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("memo_id=? AND user_id=?", in.MemoID, in.UserID).
			First(&p).Error; err != nil {
			return err
		}

		switch in.Type {
		case "UPDATED":
			p.Content = *in.Content
			p.Tags = pq.StringArray(ExtractTags(p.Content))
		case "ARCHIVED":
			p.Archived = true
		case "RESTORED":
			p.Archived = false
		case "REMINDER_SET":
			p.RemindAt = in.RemindAt
		case "REMINDER_CLEARED":
			p.RemindAt = nil
		}

		// version = last event id
		var last MemoEvent
		if err := tx.Where("memo_id=? AND user_id=?", in.MemoID, in.UserID).Order("id desc").First(&last).Error; err != nil {
			return err
		}
		p.Version = last.ID
		p.UpdatedAt = time.Now()

		if err := tx.Save(&p).Error; err != nil {
			return err
		}

		// enqueue job when REMINDER_SET (atomic)
		if in.Type == "REMINDER_SET" && in.RemindAt != nil {
			payload, _ := json.Marshal(map[string]any{"memo_id": in.MemoID})
			j := jobs.Job{
				UserID:  in.UserID,
				Type:    "REMINDER_DISPATCH",
				Payload: payload,
				RunAt:   *in.RemindAt,
				Status:  "PENDING",
			}
			if err := tx.Create(&j).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Service) insertEvent(tx *gorm.DB, memoID, userID uint64, typ string, payload map[string]any, idem *string) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ev := MemoEvent{
		MemoID:         memoID,
		UserID:         userID,
		Type:           typ,
		Payload:        json.RawMessage(b),
		IdempotencyKey: idem,
		CreatedAt:      time.Now(),
	}
	return tx.Create(&ev).Error
}
