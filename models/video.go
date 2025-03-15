package models

import "gorm.io/gorm"

type Video struct {
	gorm.Model
	Title  string `gorm:"not null"`
	UserID uint
	User   User
}
