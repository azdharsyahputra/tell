package auth

import "time"

type User struct {
	ID           uint64    `gorm:"primaryKey"`
	Email        string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    `gorm:"not null"`
	CreatedAt    time.Time `gorm:"not null;default:now()"`
}
