package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/internal/usecase"
)

type registerRequest struct {
	Email        string      `json:"email" validate:"required,email"`
	Password     string      `json:"password" validate:"required,min=8"`
	LastName     string      `json:"last_name" validate:"required"`
	FirstName    string      `json:"first_name" validate:"required"`
	Patronymic   *string     `json:"patronymic,omitempty"`
	Role         domain.Role `json:"role" validate:"required"`
	DepartmentID string      `json:"department_id" validate:"required,uuid"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Validation failed: "+err.Error())
		return
	}

	deptID, _ := uuid.Parse(req.DepartmentID)

	input := usecase.RegisterUserInput{
		Email:        req.Email,
		Password:     req.Password,
		LastName:     req.LastName,
		FirstName:    req.FirstName,
		Patronymic:   req.Patronymic,
		Role:         req.Role,
		DepartmentID: deptID,
	}

	user, err := h.authUsecase.Register(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserExists):
			h.renderError(w, r, http.StatusConflict, "User with this email already exists")
		case errors.Is(err, usecase.ErrDeptNotFound):
			h.renderError(w, r, http.StatusBadRequest, "Department not found")
		default:
			h.log.Errorf("failed to register user: %v", err)
			h.renderError(w, r, http.StatusInternalServerError, "Failed to register user")
		}
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, user)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Validation failed: "+err.Error())
		return
	}

	token, err := h.authUsecase.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidCreds) {
			h.renderError(w, r, http.StatusUnauthorized, "Invalid email or password")
			return
		}
		h.log.Errorf("failed to login: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Failed to login")
		return
	}

	render.JSON(w, r, loginResponse{Token: token})
}
