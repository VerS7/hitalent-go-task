package service

import (
	"context"
	"time"
)

type DeleteMode string

const (
	DeleteModeCascade  DeleteMode = "cascade"
	DeleteModeReassign DeleteMode = "reassign"
)

type CreateDepartmentInput struct {
	Name     string
	ParentID *uint
}

type UpdateDepartmentInput struct {
	Name        *string
	ParentIDSet bool
	ParentID    *uint
}

type CreateEmployeeInput struct {
	FullName string
	Position string
	HiredAt  *time.Time
}

type GetDepartmentOptions struct {
	Depth            int
	IncludeEmployees bool
}

type DepartmentDTO struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	ParentID  *uint     `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
}

type EmployeeDTO struct {
	ID           uint      `json:"id"`
	DepartmentID uint      `json:"department_id"`
	FullName     string    `json:"full_name"`
	Position     string    `json:"position"`
	HiredAt      *string   `json:"hired_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type DepartmentTree struct {
	Department DepartmentDTO    `json:"department"`
	Employees  *[]EmployeeDTO   `json:"employees,omitempty"`
	Children   []DepartmentTree `json:"children"`
}

type Manager interface {
	CreateDepartment(ctx context.Context, input CreateDepartmentInput) (DepartmentDTO, error)
	CreateEmployee(ctx context.Context, departmentID uint, input CreateEmployeeInput) (EmployeeDTO, error)
	GetDepartment(ctx context.Context, departmentID uint, options GetDepartmentOptions) (DepartmentTree, error)
	UpdateDepartment(ctx context.Context, departmentID uint, input UpdateDepartmentInput) (DepartmentDTO, error)
	DeleteDepartment(ctx context.Context, departmentID uint, mode DeleteMode, reassignToDepartmentID *uint) error
}
