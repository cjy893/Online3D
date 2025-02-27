package models

import "gorm.io/gorm"

type Work struct {
	gorm.Model
	UserName    string
	FileName    string
	FilePath    string
	Status      string
	ProcessTime string
	errorLog    string
}
