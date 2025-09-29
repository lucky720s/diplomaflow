package http

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/internal/middleware"
)

func (h *Handler) InitRoutes() *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID, chimiddleware.RealIP, chimiddleware.Logger, chimiddleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware)

			r.Route("/students", func(r chi.Router) {
				r.With(middleware.RoleMiddleware(domain.RoleStudent)).Get("/me", h.GetMyProfile)
			})

			r.Route("/departments", func(r chi.Router) {
				r.With(middleware.RoleMiddleware(domain.RoleSysAdmin, domain.RoleDeptAdmin)).Post("/", h.CreateDepartment)
				r.Get("/{id}", h.GetDepartmentByID)
			})
		})
	})

	return r
}
