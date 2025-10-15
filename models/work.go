package models

import "gorm.io/gorm"

type Work struct {
	gorm.Model
	WorkName    string `gorm:"not null"`
	IsPublic    bool   `gorm:"not null"`
	CoverUrl    string
	Status      string `gorm:"not null"`
	ProcessTime string
	ErrorLog    string
	Iterations  string
	UserID      uint
	User        User
}
