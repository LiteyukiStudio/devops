package model

import "time"

type SecretValue struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	CipherRef string    `gorm:"not null" json:"-"`
	CreatedBy string    `gorm:"index" json:"createdBy"`
	Resource  string    `gorm:"index" json:"resource"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
