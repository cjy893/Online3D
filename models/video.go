package models

import "gorm.io/gorm"

type Video struct {
	gorm.Model
	VideoID  string
	UserName string
	FileName string
	FilePath string
	Status   string
	ErrorLog string
}
