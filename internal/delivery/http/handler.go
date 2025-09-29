package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/pkg/logger"
)

// Интерфейсы usecase
type StudentUsecase interface {
	RegisterStudent(ctx context.Context, fullName string, departmentID string) (*domain.Student, error)
	GetStudentByID(ctx context.Context, id string) (*domain.Student, error)
	ListStudents(ctx context.Context) ([]*domain.Student, error)
	DeleteStudent(ctx context.Context, id string) error
}

type DepartmentUsecase interface {
	CreateDepartment(ctx context.Context, name, universityID string) (*domain.Department, error)
	GetDepartmentByID(ctx context.Context, id string) (*domain.Department, error)
}

// Handler содержит зависимости
type Handler struct {
	log               *logger.Logger
	studentUsecase    StudentUsecase
	departmentUsecase DepartmentUsecase
	validator         *validator.Validate
}

// Конструктор
func NewHandler(log *logger.Logger, studentUC StudentUsecase, departmentUC DepartmentUsecase) *Handler {
	return &Handler{
		log:               log,
		studentUsecase:    studentUC,
		departmentUsecase: departmentUC,
		validator:         validator.New(),
	}
}

// Request/Response структуры
type registerStudentRequest struct {
	FullName     string `json:"full_name" validate:"required,min=2"`
	DepartmentID string `json:"department_id" validate:"required"`
}

type createDepartmentRequest struct {
	Name         string `json:"name" validate:"required,min=5"`
	UniversityID string `json:"university_id" validate:"required"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- Студенты ---
func (h *Handler) registerStudent(w http.ResponseWriter, r *http.Request) {
	var req registerStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Errorf("failed to decode request body: %v", err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Error: "Invalid request body"})
		return
	}
	if err := h.validator.Struct(req); err != nil {
		h.log.Warnf("validation failed: %v", err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Error: "Invalid request data: " + err.Error()})
		return
	}

	student, err := h.studentUsecase.RegisterStudent(r.Context(), req.FullName, req.DepartmentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, errorResponse{Error: "Invalid department_id: department does not exist"})
			return
		}
		h.log.Errorf("failed to register student: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Error: "Internal server error"})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, student)
}

func (h *Handler) getStudentByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	student, err := h.studentUsecase.GetStudentByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, errorResponse{Error: "Student not found"})
			return
		}
		h.log.Errorf("failed to get student: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Error: "Internal server error"})
		return
	}
	render.JSON(w, r, student)
}

func (h *Handler) listStudents(w http.ResponseWriter, r *http.Request) {
	students, err := h.studentUsecase.ListStudents(r.Context())
	if err != nil {
		h.log.Errorf("failed to list students: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Error: "Internal server error"})
		return
	}
	render.JSON(w, r, students)
}

func (h *Handler) deleteStudent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := h.studentUsecase.DeleteStudent(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, errorResponse{Error: "Student not found"})
			return
		}
		h.log.Errorf("failed to delete student: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Error: "Internal server error"})
		return
	}
	render.Status(r, http.StatusNoContent)
}

// --- Кафедры ---
func (h *Handler) createDepartment(w http.ResponseWriter, r *http.Request) {
	var req createDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Error: "Invalid request body"})
		return
	}
	if err := h.validator.Struct(req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Error: "Invalid request data: " + err.Error()})
		return
	}
	dept, err := h.departmentUsecase.CreateDepartment(r.Context(), req.Name, req.UniversityID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Error: "Internal server error"})
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, dept)
}

func (h *Handler) getDepartmentByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dept, err := h.departmentUsecase.GetDepartmentByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, errorResponse{Error: "Department not found"})
			return
		}
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Error: "Internal server error"})
		return
	}
	render.JSON(w, r, dept)
}
