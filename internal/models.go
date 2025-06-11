package internal

import (
	"time"
)

type User struct {
	ID    uint   `gorm:"primaryKey"`
	Email string `gorm:"uniqueIndex;not null"`
	Role  string `gorm:"not null"`
}

type Movie struct {
	ID          uint   `gorm:"primaryKey"`
	Title       string `gorm:"not null"`
	Description string
	VideoURL    string
	PosterURL   string
	ObjectName  string // имя объекта в MinIO
}

type WatchHistory struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"index"`
	MovieID   uint `gorm:"index"`
	Progress  int  // seconds
	LastWatch time.Time
}
