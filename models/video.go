package models

import "gorm.io/gorm"

type Video struct {
	gorm.Model
	Title    string `gorm:"not null"`
	IsPublic bool   `gorm:"not null"`
	CoverUrl string
	UserID   uint
	User     User
}
