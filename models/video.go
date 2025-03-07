package models

import "gorm.io/gorm"

type Video struct {
	gorm.Model
	Title    string
	FilePath string
	Status   string
	ErrorLog string
	UserID   uint
	Works    []Work
}
