package model

import "time"

type WorkerTaskEvent struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	TaskID      string    `gorm:"index;not null" json:"taskId"`
	TaskType    string    `gorm:"index;not null" json:"taskType"`
	DedupeKey   string    `gorm:"index;not null" json:"dedupeKey"`
	ActorID     string    `gorm:"index" json:"actorId"`
	ResourceRef string    `gorm:"index" json:"resourceRef"`
	Status      string    `gorm:"index;not null" json:"status"`
	Message     string    `json:"message"`
	Attempt     int       `json:"attempt"`
	CreatedAt   time.Time `json:"createdAt"`
}
