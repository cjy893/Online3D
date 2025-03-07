package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Account  string `gorm:"unique"`
	Password string `gorm:"not null"`
	Email    string `gorm:"unique"`
	Videos   []Video
	Works    []Work
}
