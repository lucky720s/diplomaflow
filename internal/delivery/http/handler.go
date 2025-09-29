// file: internal/delivery/http/handler.go
package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/pkg/logger"
)

// Определяем интерфейс для usecase, от которого будет зависеть наш хендлер.
// Это позволяет нам не зависеть от конкретной реализации usecase.
type StudentUsecase interface {
	RegisterStudent(ctx context.Context, fullName, departmentID string) (*domain.Student, error)
}

type Handler struct {
	log            *logger.Logger
	studentUsecase StudentUsecase
}

func NewHandler(log *logger.Logger, studentUC StudentUsecase) *Handler {
	return &Handler{
		log:            log,
		studentUsecase: studentUC,
	}
}

// Структура для входящего запроса на регистрацию студента
type registerStudentRequest struct {
	FullName     string `json:"full_name"`
	DepartmentID string `json:"department_id"`
}

// Структура для ответа об ошибке
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

	student, err := h.studentUsecase.RegisterStudent(r.Context(), req.FullName, req.DepartmentID)
	if err != nil {
		// Здесь можно будет проверять тип ошибки и отдавать разные статусы
		h.log.Errorf("failed to register student: %v", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Error: "Internal server error"})
		return
	}

	// Отправляем успешный ответ
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, student)
}
