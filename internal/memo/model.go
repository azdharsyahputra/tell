package memo

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// Memo is a container. State is derived from events and stored in projection.
type Memo struct {
	ID        uint64    `gorm:"primaryKey"`
	UserID    uint64    `gorm:"index;not null"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
}

// MemoEvent is append-only.
// Use IdempotencyKey to prevent duplicates per user (optional header).
type MemoEvent struct {
	ID             uint64          `gorm:"primaryKey"`
	MemoID         uint64          `gorm:"index;not null"`
	UserID         uint64          `gorm:"index;not null"`
	Type           string          `gorm:"not null"`
	Payload        json.RawMessage `gorm:"type:jsonb;not null;default:'{}'::jsonb"`
	IdempotencyKey *string         `gorm:"index"`
	CreatedAt      time.Time       `gorm:"not null;default:now()"`
}

// MemoProjection is the current state for fast read/search.
type MemoProjection struct {
	MemoID   uint64     `gorm:"primaryKey"`
	UserID   uint64     `gorm:"index;not null"`
	Content  string     `gorm:"type:text;not null;default:''"`
	Archived bool       `gorm:"not null;default:false"`
	RemindAt *time.Time `gorm:"type:timestamptz"`

	Tags pq.StringArray `gorm:"type:text[];not null;default:'{}'"`

	Version   uint64    `gorm:"not null;default:0"`
	UpdatedAt time.Time `gorm:"index;not null;default:now()"`
}

// Tag is a normalized hashtag per user.
type Tag struct {
	ID        uint64    `gorm:"primaryKey"`
	UserID    uint64    `gorm:"index;not null"`
	Name      string    `gorm:"index;not null"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
}

// MemoTag is join table for analytics/list tags.
type MemoTag struct {
	MemoID uint64 `gorm:"primaryKey"`
	UserID uint64 `gorm:"index;not null"`
	TagID  uint64 `gorm:"primaryKey"`
}
