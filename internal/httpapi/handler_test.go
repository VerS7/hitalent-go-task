package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hitalent-go-task/internal/service"
)

type stubService struct {
	createDepartmentFn func(ctx context.Context, input service.CreateDepartmentInput) (service.DepartmentDTO, error)
	createEmployeeFn   func(ctx context.Context, departmentID uint, input service.CreateEmployeeInput) (service.EmployeeDTO, error)
	getDepartmentFn    func(ctx context.Context, departmentID uint, options service.GetDepartmentOptions) (service.DepartmentTree, error)
	updateDepartmentFn func(ctx context.Context, departmentID uint, input service.UpdateDepartmentInput) (service.DepartmentDTO, error)
	deleteDepartmentFn func(ctx context.Context, departmentID uint, mode service.DeleteMode, reassignToDepartmentID *uint) error
}

func (s stubService) CreateDepartment(ctx context.Context, input service.CreateDepartmentInput) (service.DepartmentDTO, error) {
	if s.createDepartmentFn == nil {
		return service.DepartmentDTO{}, nil
	}
	return s.createDepartmentFn(ctx, input)
}

func (s stubService) CreateEmployee(ctx context.Context, departmentID uint, input service.CreateEmployeeInput) (service.EmployeeDTO, error) {
	if s.createEmployeeFn == nil {
		return service.EmployeeDTO{}, nil
	}
	return s.createEmployeeFn(ctx, departmentID, input)
}

func (s stubService) GetDepartment(ctx context.Context, departmentID uint, options service.GetDepartmentOptions) (service.DepartmentTree, error) {
	if s.getDepartmentFn == nil {
		return service.DepartmentTree{}, nil
	}
	return s.getDepartmentFn(ctx, departmentID, options)
}

func (s stubService) UpdateDepartment(ctx context.Context, departmentID uint, input service.UpdateDepartmentInput) (service.DepartmentDTO, error) {
	if s.updateDepartmentFn == nil {
		return service.DepartmentDTO{}, nil
	}
	return s.updateDepartmentFn(ctx, departmentID, input)
}

func (s stubService) DeleteDepartment(ctx context.Context, departmentID uint, mode service.DeleteMode, reassignToDepartmentID *uint) error {
	if s.deleteDepartmentFn == nil {
		return nil
	}
	return s.deleteDepartmentFn(ctx, departmentID, mode, reassignToDepartmentID)
}

func TestCreateDepartment(t *testing.T) {
	handler := NewHandler(stubService{
		createDepartmentFn: func(ctx context.Context, input service.CreateDepartmentInput) (service.DepartmentDTO, error) {
			if input.Name != "Backend" {
				t.Fatalf("unexpected department name: %s", input.Name)
			}
			return service.DepartmentDTO{
				ID:        1,
				Name:      "Backend",
				ParentID:  nil,
				CreatedAt: time.Now(),
			}, nil
		},
	}, log.New(io.Discard, "", 0))

	body := bytes.NewBufferString(`{"name":"Backend"}`)
	req := httptest.NewRequest(http.MethodPost, "/departments", body)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if payload["name"] != "Backend" {
		t.Fatalf("expected department name Backend, got %v", payload["name"])
	}
}

func TestGetDepartmentDepthValidation(t *testing.T) {
	handler := NewHandler(stubService{}, log.New(io.Discard, "", 0))

	req := httptest.NewRequest(http.MethodGet, "/departments/1?depth=6", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}
