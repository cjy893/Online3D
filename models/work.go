package models

import "gorm.io/gorm"

type Work struct {
	gorm.Model
	WorkID   string
	UserName string
	FileName string
	FilePath string
}
