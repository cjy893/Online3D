package models

import "gorm.io/gorm"

type Video struct {
	gorm.Model
	VideoID     string `gorm:"unique"`
	UserName    string
	FileName    string
	FilePath    string
	Status      string
	ErrorLog    string
	ProcessTime string
}
