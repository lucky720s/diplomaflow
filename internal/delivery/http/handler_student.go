package http

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"github.com/lucky720s/diplomaflow/internal/middleware"
)

func (h *Handler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserContextKey).(*middleware.UserContext)
	if !ok {
		h.renderError(w, r, http.StatusUnauthorized, "Unauthorized")
		return
	}

	profile, err := h.studentUsecase.GetProfile(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.renderError(w, r, http.StatusNotFound, "Student profile not found")
			return
		}
		h.log.Errorf("failed to get student profile: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Failed to get profile")
		return
	}

	render.JSON(w, r, profile)
}
