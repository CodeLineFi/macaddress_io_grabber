package database

import (
	"time"
)

type CustomNote struct {
	ID         uint64 `gorm:"primary_key"`
	Assignment string `gorm:"column:assignment;type:varchar(100);not null;index:assignment_idx"`
	Note       string `gorm:"column:note;type:text;not null"`
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
}

func (customNote CustomNote) TableName() string {
	return "custom_notes"
}
