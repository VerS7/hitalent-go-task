package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"hitalent-go-task/internal/apperror"
	"hitalent-go-task/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type DepartmentService struct {
	db *gorm.DB
}

func NewDepartmentService(db *gorm.DB) *DepartmentService {
	return &DepartmentService{db: db}
}

func (s *DepartmentService) CreateDepartment(ctx context.Context, input CreateDepartmentInput) (DepartmentDTO, error) {
	name, err := normalizeRequiredString(input.Name, "name")
	if err != nil {
		return DepartmentDTO{}, err
	}

	if input.ParentID != nil {
		if err := s.ensureDepartmentExists(ctx, *input.ParentID); err != nil {
			return DepartmentDTO{}, err
		}
	}

	exists, err := s.siblingNameExists(ctx, input.ParentID, name, nil)
	if err != nil {
		return DepartmentDTO{}, err
	}
	if exists {
		return DepartmentDTO{}, apperror.New(apperror.CodeConflict, "department name must be unique under the same parent")
	}

	department := models.Department{
		Name:     name,
		ParentID: input.ParentID,
	}

	if err := s.db.WithContext(ctx).Create(&department).Error; err != nil {
		return DepartmentDTO{}, mapDatabaseError(err)
	}

	return departmentToDTO(department), nil
}

func (s *DepartmentService) CreateEmployee(ctx context.Context, departmentID uint, input CreateEmployeeInput) (EmployeeDTO, error) {
	fullName, err := normalizeRequiredString(input.FullName, "full_name")
	if err != nil {
		return EmployeeDTO{}, err
	}

	position, err := normalizeRequiredString(input.Position, "position")
	if err != nil {
		return EmployeeDTO{}, err
	}

	if err := s.ensureDepartmentExists(ctx, departmentID); err != nil {
		return EmployeeDTO{}, err
	}

	employee := models.Employee{
		DepartmentID: departmentID,
		FullName:     fullName,
		Position:     position,
		HiredAt:      input.HiredAt,
	}

	if err := s.db.WithContext(ctx).Create(&employee).Error; err != nil {
		return EmployeeDTO{}, mapDatabaseError(err)
	}

	return employeeToDTO(employee), nil
}

func (s *DepartmentService) GetDepartment(ctx context.Context, departmentID uint, options GetDepartmentOptions) (DepartmentTree, error) {
	var department models.Department
	if err := s.db.WithContext(ctx).First(&department, departmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DepartmentTree{}, apperror.New(apperror.CodeNotFound, "department not found")
		}
		return DepartmentTree{}, fmt.Errorf("load department: %w", err)
	}

	if options.Depth < 0 || options.Depth > 5 {
		return DepartmentTree{}, apperror.New(apperror.CodeValidation, "depth must be between 0 and 5")
	}

	tree, err := s.buildTree(ctx, department, options.Depth, options.IncludeEmployees)
	if err != nil {
		return DepartmentTree{}, err
	}

	return tree, nil
}

func (s *DepartmentService) UpdateDepartment(ctx context.Context, departmentID uint, input UpdateDepartmentInput) (DepartmentDTO, error) {
	var department models.Department
	if err := s.db.WithContext(ctx).First(&department, departmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DepartmentDTO{}, apperror.New(apperror.CodeNotFound, "department not found")
		}
		return DepartmentDTO{}, fmt.Errorf("load department: %w", err)
	}

	if input.Name == nil && !input.ParentIDSet {
		return departmentToDTO(department), nil
	}

	newName := department.Name
	if input.Name != nil {
		normalized, err := normalizeRequiredString(*input.Name, "name")
		if err != nil {
			return DepartmentDTO{}, err
		}
		newName = normalized
	}

	newParentID := department.ParentID
	if input.ParentIDSet {
		newParentID = input.ParentID
		if newParentID != nil && *newParentID == departmentID {
			return DepartmentDTO{}, apperror.New(apperror.CodeValidation, "department cannot be parent of itself")
		}
	}

	if input.ParentIDSet && newParentID != nil {
		if err := s.ensureDepartmentExists(ctx, *newParentID); err != nil {
			return DepartmentDTO{}, err
		}
		willCycle, err := s.wouldCreateCycle(ctx, departmentID, *newParentID)
		if err != nil {
			return DepartmentDTO{}, err
		}
		if willCycle {
			return DepartmentDTO{}, apperror.New(apperror.CodeConflict, "department cycle detected")
		}
	}

	exists, err := s.siblingNameExists(ctx, newParentID, newName, &departmentID)
	if err != nil {
		return DepartmentDTO{}, err
	}
	if exists {
		return DepartmentDTO{}, apperror.New(apperror.CodeConflict, "department name must be unique under the same parent")
	}

	updates := map[string]interface{}{}
	if input.Name != nil && newName != department.Name {
		updates["name"] = newName
	}
	if input.ParentIDSet && !equalUintPtr(department.ParentID, newParentID) {
		updates["parent_id"] = newParentID
	}

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&department).Updates(updates).Error; err != nil {
			return DepartmentDTO{}, mapDatabaseError(err)
		}
		if err := s.db.WithContext(ctx).First(&department, departmentID).Error; err != nil {
			return DepartmentDTO{}, fmt.Errorf("reload department: %w", err)
		}
	}

	return departmentToDTO(department), nil
}

func (s *DepartmentService) DeleteDepartment(ctx context.Context, departmentID uint, mode DeleteMode, reassignToDepartmentID *uint) error {
	var department models.Department
	if err := s.db.WithContext(ctx).First(&department, departmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.New(apperror.CodeNotFound, "department not found")
		}
		return fmt.Errorf("load department: %w", err)
	}

	switch mode {
	case DeleteModeCascade:
		if err := s.db.WithContext(ctx).Delete(&models.Department{}, departmentID).Error; err != nil {
			return mapDatabaseError(err)
		}
		return nil

	case DeleteModeReassign:
		if reassignToDepartmentID == nil {
			return apperror.New(apperror.CodeValidation, "reassign_to_department_id is required when mode=reassign")
		}
		if *reassignToDepartmentID == departmentID {
			return apperror.New(apperror.CodeValidation, "reassign_to_department_id cannot be the same department")
		}
		if err := s.ensureDepartmentExists(ctx, *reassignToDepartmentID); err != nil {
			return err
		}

		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.Employee{}).
				Where("department_id = ?", departmentID).
				Update("department_id", *reassignToDepartmentID).Error; err != nil {
				return mapDatabaseError(err)
			}

			if err := tx.Model(&models.Department{}).
				Where("parent_id = ?", departmentID).
				Update("parent_id", department.ParentID).Error; err != nil {
				return mapDatabaseError(err)
			}

			if err := tx.Delete(&models.Department{}, departmentID).Error; err != nil {
				return mapDatabaseError(err)
			}

			return nil
		})

	default:
		return apperror.New(apperror.CodeValidation, "mode must be one of: cascade, reassign")
	}
}

func (s *DepartmentService) buildTree(ctx context.Context, department models.Department, depth int, includeEmployees bool) (DepartmentTree, error) {
	result := DepartmentTree{
		Department: departmentToDTO(department),
		Children:   []DepartmentTree{},
	}

	if includeEmployees {
		var employees []models.Employee
		if err := s.db.WithContext(ctx).
			Where("department_id = ?", department.ID).
			Order("full_name ASC").
			Find(&employees).Error; err != nil {
			return DepartmentTree{}, fmt.Errorf("load employees: %w", err)
		}

		employeesDTO := make([]EmployeeDTO, 0, len(employees))
		for _, employee := range employees {
			employeesDTO = append(employeesDTO, employeeToDTO(employee))
		}
		result.Employees = &employeesDTO
	}

	if depth == 0 {
		return result, nil
	}

	var children []models.Department
	if err := s.db.WithContext(ctx).
		Where("parent_id = ?", department.ID).
		Order("name ASC").
		Find(&children).Error; err != nil {
		return DepartmentTree{}, fmt.Errorf("load child departments: %w", err)
	}

	for _, child := range children {
		childTree, err := s.buildTree(ctx, child, depth-1, includeEmployees)
		if err != nil {
			return DepartmentTree{}, err
		}
		result.Children = append(result.Children, childTree)
	}

	return result, nil
}

func (s *DepartmentService) ensureDepartmentExists(ctx context.Context, departmentID uint) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Department{}).Where("id = ?", departmentID).Count(&count).Error; err != nil {
		return fmt.Errorf("check department existence: %w", err)
	}
	if count == 0 {
		return apperror.New(apperror.CodeNotFound, "department not found")
	}
	return nil
}

func (s *DepartmentService) siblingNameExists(ctx context.Context, parentID *uint, name string, excludeID *uint) (bool, error) {
	query := s.db.WithContext(ctx).Model(&models.Department{}).Where("LOWER(name) = LOWER(?)", name)
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}
	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("check sibling uniqueness: %w", err)
	}
	return count > 0, nil
}

func (s *DepartmentService) wouldCreateCycle(ctx context.Context, departmentID uint, newParentID uint) (bool, error) {
	currentID := &newParentID
	for currentID != nil {
		if *currentID == departmentID {
			return true, nil
		}

		var department models.Department
		if err := s.db.WithContext(ctx).
			Select("id", "parent_id").
			First(&department, *currentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return false, fmt.Errorf("load parent chain: %w", err)
		}
		currentID = department.ParentID
	}

	return false, nil
}

func departmentToDTO(department models.Department) DepartmentDTO {
	return DepartmentDTO{
		ID:        department.ID,
		Name:      department.Name,
		ParentID:  department.ParentID,
		CreatedAt: department.CreatedAt,
	}
}

func employeeToDTO(employee models.Employee) EmployeeDTO {
	var hiredAt *string
	if employee.HiredAt != nil {
		formatted := employee.HiredAt.Format("2006-01-02")
		hiredAt = &formatted
	}

	return EmployeeDTO{
		ID:           employee.ID,
		DepartmentID: employee.DepartmentID,
		FullName:     employee.FullName,
		Position:     employee.Position,
		HiredAt:      hiredAt,
		CreatedAt:    employee.CreatedAt,
	}
}

func normalizeRequiredString(raw string, field string) (string, error) {
	value := strings.TrimSpace(raw)
	length := utf8.RuneCountInString(value)
	if length < 1 || length > 200 {
		return "", apperror.New(apperror.CodeValidation, fmt.Sprintf("%s length must be in range 1..200", field))
	}
	return value, nil
}

func equalUintPtr(a *uint, b *uint) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func mapDatabaseError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return apperror.New(apperror.CodeConflict, "resource with the same unique attributes already exists")
		}
		if pgErr.Code == "23503" {
			return apperror.New(apperror.CodeValidation, "invalid foreign key reference")
		}
	}
	return err
}
