package models

import "gorm.io/gorm"

type Video struct {
	gorm.Model
	UserName string
	Title    string
	FilePath string
	Status   string
	ErrorLog string
}
