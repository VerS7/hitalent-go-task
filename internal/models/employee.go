package models

import "time"

type Employee struct {
	ID           uint       `gorm:"primaryKey"`
	DepartmentID uint       `gorm:"not null;index"`
	Department   Department `gorm:"foreignKey:DepartmentID"`
	FullName     string     `gorm:"type:varchar(200);not null"`
	Position     string     `gorm:"type:varchar(200);not null"`
	HiredAt      *time.Time `gorm:"type:date"`
	CreatedAt    time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
}
