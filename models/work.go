package models

import "gorm.io/gorm"

type Work struct {
	gorm.Model
	WorkName    string
	FilePath    string
	Status      string
	ProcessTime string
	ErrorLog    string
	Iterations  string
	UserID      uint
	VideoID     uint
}
