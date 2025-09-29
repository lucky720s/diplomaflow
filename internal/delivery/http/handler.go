package http

import (
	"context"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/internal/usecase"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"net/http"
)

type AuthUsecase interface {
	Register(ctx context.Context, input usecase.RegisterUserInput) (*domain.User, error)
	Login(ctx context.Context, email, password string) (string, error)
}

type StudentUsecase interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*domain.StudentProfile, error)
}

type DepartmentUsecase interface {
	Create(ctx context.Context, name, universityIDStr string) (*domain.Department, error)
	GetByID(ctx context.Context, idStr string) (*domain.Department, error)
}

type Handler struct {
	log               *logger.Logger
	authUsecase       AuthUsecase
	studentUsecase    StudentUsecase
	departmentUsecase DepartmentUsecase
	validator         *validator.Validate
}

func NewHandler(log *logger.Logger, authUC AuthUsecase, studentUC StudentUsecase, deptUC DepartmentUsecase) *Handler {
	return &Handler{
		log:               log,
		authUsecase:       authUC,
		studentUsecase:    studentUC,
		departmentUsecase: deptUC,
		validator:         validator.New(),
	}
}

type errorResponse struct {
	Message string `json:"message"`
}

func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	render.Status(r, status)
	render.JSON(w, r, errorResponse{Message: message})
}
