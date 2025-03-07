package models

import "gorm.io/gorm"

type Video struct {
	gorm.Model
	Title    string `gorm:"not null"`
	FilePath string `gorm:"not null"`
	Status   string `gorm:"not null"`
	ErrorLog string
	UserID   uint
	Works    []Work
}
