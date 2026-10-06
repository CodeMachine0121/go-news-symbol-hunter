package entities

import "time"

type ApiKey struct {
	ID         uint   `gorm:"primaryKey"`
	Name       string `gorm:"size:100;not null"`
	SecretHash string `gorm:"size:64;not null;uniqueIndex"`
	IsActive   bool   `gorm:"not null;default:false"`
	RevokedAt  *time.Time
	CreatedAt  time.Time
}
