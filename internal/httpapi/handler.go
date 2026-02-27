package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hitalent-go-task/internal/apperror"
	"hitalent-go-task/internal/service"
)

type Handler struct {
	service service.Manager
	logger  *log.Logger
}

func NewHandler(svc service.Manager, logger *log.Logger) *Handler {
	return &Handler{
		service: svc,
		logger:  logger,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 0 || parts[0] != "departments" {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}

	switch {
	case len(parts) == 1:
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.handleCreateDepartment(w, r)
		return

	case len(parts) == 2:
		departmentID, err := parseUintID(parts[1])
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid department id")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.handleGetDepartment(w, r, departmentID)
		case http.MethodPatch:
			h.handleUpdateDepartment(w, r, departmentID)
		case http.MethodDelete:
			h.handleDeleteDepartment(w, r, departmentID)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return

	case len(parts) == 3 && parts[2] == "employees":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		departmentID, err := parseUintID(parts[1])
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid department id")
			return
		}

		h.handleCreateEmployee(w, r, departmentID)
		return
	}

	writeError(w, http.StatusNotFound, "route not found")
}

type createDepartmentRequest struct {
	Name     string `json:"name"`
	ParentID *uint  `json:"parent_id"`
}

type createEmployeeRequest struct {
	FullName string  `json:"full_name"`
	Position string  `json:"position"`
	HiredAt  *string `json:"hired_at"`
}

type updateDepartmentRequest struct {
	Name     *string      `json:"name"`
	ParentID optionalUint `json:"parent_id"`
}

func (h *Handler) handleCreateDepartment(w http.ResponseWriter, r *http.Request) {
	var req createDepartmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	department, err := h.service.CreateDepartment(r.Context(), service.CreateDepartmentInput{
		Name:     req.Name,
		ParentID: req.ParentID,
	})
	if err != nil {
		h.respondWithError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, department)
}

func (h *Handler) handleCreateEmployee(w http.ResponseWriter, r *http.Request, departmentID uint) {
	var req createEmployeeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hiredAt, err := parseDate(req.HiredAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	employee, err := h.service.CreateEmployee(r.Context(), departmentID, service.CreateEmployeeInput{
		FullName: req.FullName,
		Position: req.Position,
		HiredAt:  hiredAt,
	})
	if err != nil {
		h.respondWithError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, employee)
}

func (h *Handler) handleGetDepartment(w http.ResponseWriter, r *http.Request, departmentID uint) {
	options, err := parseGetDepartmentOptions(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.service.GetDepartment(r.Context(), departmentID, options)
	if err != nil {
		h.respondWithError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleUpdateDepartment(w http.ResponseWriter, r *http.Request, departmentID uint) {
	var req updateDepartmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	updatedDepartment, err := h.service.UpdateDepartment(r.Context(), departmentID, service.UpdateDepartmentInput{
		Name:        req.Name,
		ParentIDSet: req.ParentID.Set,
		ParentID:    req.ParentID.Value,
	})
	if err != nil {
		h.respondWithError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updatedDepartment)
}

func (h *Handler) handleDeleteDepartment(w http.ResponseWriter, r *http.Request, departmentID uint) {
	mode := service.DeleteMode(strings.TrimSpace(strings.ToLower(r.URL.Query().Get("mode"))))
	reassignToDepartmentID, err := parseOptionalReassignDepartmentID(r.URL.Query().Get("reassign_to_department_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.DeleteDepartment(r.Context(), departmentID, mode, reassignToDepartmentID); err != nil {
		h.respondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) respondWithError(w http.ResponseWriter, err error) {
	switch apperror.GetCode(err) {
	case apperror.CodeValidation:
		writeError(w, http.StatusBadRequest, err.Error())
	case apperror.CodeNotFound:
		writeError(w, http.StatusNotFound, err.Error())
	case apperror.CodeConflict:
		writeError(w, http.StatusConflict, err.Error())
	default:
		h.logger.Printf("unexpected error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func decodeJSON(r *http.Request, target interface{}) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("invalid JSON body")
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("invalid JSON body")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func parseUintID(raw string) (uint, error) {
	id64, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id64 == 0 {
		return 0, errors.New("invalid id")
	}
	return uint(id64), nil
}

func parseGetDepartmentOptions(r *http.Request) (service.GetDepartmentOptions, error) {
	query := r.URL.Query()

	depth := 1
	if rawDepth := strings.TrimSpace(query.Get("depth")); rawDepth != "" {
		parsedDepth, err := strconv.Atoi(rawDepth)
		if err != nil {
			return service.GetDepartmentOptions{}, errors.New("depth must be an integer")
		}
		if parsedDepth < 0 || parsedDepth > 5 {
			return service.GetDepartmentOptions{}, errors.New("depth must be between 0 and 5")
		}
		depth = parsedDepth
	}

	includeEmployees := true
	if rawIncludeEmployees := strings.TrimSpace(query.Get("include_employees")); rawIncludeEmployees != "" {
		parsedIncludeEmployees, err := strconv.ParseBool(rawIncludeEmployees)
		if err != nil {
			return service.GetDepartmentOptions{}, errors.New("include_employees must be a boolean")
		}
		includeEmployees = parsedIncludeEmployees
	}

	return service.GetDepartmentOptions{
		Depth:            depth,
		IncludeEmployees: includeEmployees,
	}, nil
}

func parseOptionalReassignDepartmentID(raw string) (*uint, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	parsedID, err := parseUintID(value)
	if err != nil {
		return nil, errors.New("reassign_to_department_id must be a positive integer")
	}
	return &parsedID, nil
}

func parseDate(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}

	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, errors.New("hired_at must be in YYYY-MM-DD format")
	}

	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, errors.New("hired_at must be in YYYY-MM-DD format")
	}

	return &parsed, nil
}

type optionalUint struct {
	Set   bool
	Value *uint
}

func (o *optionalUint) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		o.Value = nil
		return nil
	}

	var value uint
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}
