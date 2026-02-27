package models

import "time"

type Department struct {
	ID        uint         `gorm:"primaryKey"`
	Name      string       `gorm:"type:varchar(200);not null"`
	ParentID  *uint        `gorm:"index"`
	Parent    *Department  `gorm:"foreignKey:ParentID;references:ID"`
	Children  []Department `gorm:"foreignKey:ParentID;references:ID"`
	Employees []Employee   `gorm:"foreignKey:DepartmentID;references:ID"`
	CreatedAt time.Time    `gorm:"not null;default:CURRENT_TIMESTAMP"`
}
