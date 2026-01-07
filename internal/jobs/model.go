package jobs

import "time"

type Job struct {
	ID     uint64 `gorm:"primaryKey"`
	UserID uint64 `gorm:"index;not null"`

	Type    string `gorm:"type:text;not null"` // REMINDER_DISPATCH
	Payload []byte `gorm:"type:jsonb;not null;default:'{}'::jsonb"`

	RunAt  time.Time `gorm:"index;not null"`
	Status string    `gorm:"index;not null;default:'PENDING'"` // PENDING/RUNNING/DONE/FAILED/CANCELLED

	Attempts    int `gorm:"not null;default:0"`
	MaxAttempts int `gorm:"not null;default:8"`

	LockedBy *string    `gorm:"type:text"`
	LockedAt *time.Time `gorm:"type:timestamptz"`

	LastError *string `gorm:"type:text"`

	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}
