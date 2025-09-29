package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors" // <-- ИМПОРТИРУЕМ СТАНДАРТНЫЙ ПАКЕТ GO
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/pkg/logger"
)

// StudentUsecase определяет интерфейс, от которого зависит хендлер
type StudentUsecase interface {
	RegisterStudent(ctx context.Context, fullName, departmentID string) (*domain.Student, error)
	GetStudentByID(ctx context.Context, id string) (*domain.Student, error)
	ListStudents(ctx context.Context) ([]*domain.Student, error)
	DeleteStudent(ctx context.Context, id string) error
	GetStudentWithDetails(ctx context.Context, id string) (*domain.StudentWithDepartment, error) // <-- ДОБАВИТЬ

}
type DepartmentUsecase interface {
	CreateDepartment(ctx context.Context, name, universityID string) (*domain.Department, error)
	GetDepartmentByID(ctx context.Context, id string) (*domain.Department, error)
}

type Handler struct {
	log               *logger.Logger
	studentUsecase    StudentUsecase
	departmentUsecase DepartmentUsecase
	validator         *validator.Validate
}

func NewHandler(log *logger.Logger, studentUC StudentUsecase, departmentUC DepartmentUsecase) *Handler {
	return &Handler{
		log:               log,
		studentUsecase:    studentUC,
		departmentUsecase: departmentUC,
		validator:         validator.New(),
	}
}

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

// registerStudent обрабатывает POST /students
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
		// Теперь errors.Is() ссылается на стандартную библиотеку и работает правильно
		if errors.Is(err, sql.ErrNoRows) {
			h.log.Warnf("failed to register student, department not found: %v", err)
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

// getStudentByID обрабатывает GET /students/{id}
// ОБНОВЛЕННЫЙ ХЕНДЛЕР
func (h *Handler) getStudentByID(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "id")

	// Вызываем новый метод usecase
	student, err := h.studentUsecase.GetStudentWithDetails(r.Context(), studentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.log.Warnf("student with id %s not found: %v", studentID, err)
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, errorResponse{Error: "Student not found"})
			return
		}

		h.log.Errorf("failed to get student by id %s: %v", studentID, err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Error: "Internal server error"})
		return
	}

	render.JSON(w, r, student)
}

// listStudents обрабатывает GET /students
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

// deleteStudent обрабатывает DELETE /students/{id}
func (h *Handler) deleteStudent(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "id")

	err := h.studentUsecase.DeleteStudent(r.Context(), studentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.log.Warnf("student with id %s not found for deletion: %v", studentID, err)
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, errorResponse{Error: "Student not found"})
			return
		}

		h.log.Errorf("failed to delete student %s: %v", studentID, err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Error: "Internal server error"})
		return
	}

	render.Status(r, http.StatusNoContent)
}

func (h *Handler) createDepartment(w http.ResponseWriter, r *http.Request) {
	var req createDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	dept, err := h.departmentUsecase.CreateDepartment(r.Context(), req.Name, req.UniversityID)
	if err != nil {
		h.log.Errorf("failed to create department: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Error: "Internal server error"})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, dept)
}

func (h *Handler) getDepartmentByID(w http.ResponseWriter, r *http.Request) {
	deptID := chi.URLParam(r, "id")
	dept, err := h.departmentUsecase.GetDepartmentByID(r.Context(), deptID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.log.Warnf("department with id %s not found: %v", deptID, err)
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, errorResponse{Error: "Department not found"})
			return
		}

		h.log.Errorf("failed to get department by id %s: %v", deptID, err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Error: "Internal server error"})
		return
	}
	render.JSON(w, r, dept)
}
