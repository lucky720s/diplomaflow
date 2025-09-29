package http

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type createDepartmentRequest struct {
	Name         string `json:"name" validate:"required,min=5"`
	UniversityID string `json:"university_id" validate:"required,uuid"`
}

func (h *Handler) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	var req createDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		h.renderError(w, r, http.StatusBadRequest, "Validation failed: "+err.Error())
		return
	}

	dept, err := h.departmentUsecase.Create(r.Context(), req.Name, req.UniversityID)
	if err != nil {
		h.log.Errorf("failed to create department: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Failed to create department")
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, dept)
}

func (h *Handler) GetDepartmentByID(w http.ResponseWriter, r *http.Request) {
	deptID := chi.URLParam(r, "id")

	dept, err := h.departmentUsecase.GetByID(r.Context(), deptID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.renderError(w, r, http.StatusNotFound, "Department not found")
			return
		}
		h.log.Errorf("failed to get department by id %s: %v", deptID, err)
		h.renderError(w, r, http.StatusInternalServerError, "Failed to get department")
		return
	}

	render.JSON(w, r, dept)
}
