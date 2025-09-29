// file: internal/delivery/http/server.go
package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (h *Handler) InitRoutes() *chi.Mux {
	r := chi.NewRouter()

	// Подключаем middleware: логгер, восстановление после паник
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger) // Логгер от chi
	r.Use(middleware.Recoverer)

	// Группа роутов для /api/v1
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/students", func(r chi.Router) {
			r.Post("/", h.registerStudent)
			r.Get("/", h.listStudents)
			r.Get("/{id}", h.getStudentByID)
			r.Delete("/{id}", h.deleteStudent)
			// TODO: r.Put("/{id}", h.updateStudent)
		})
		r.Route("/departments", func(r chi.Router) {
			r.Post("/", h.createDepartment)
			r.Get("/{id}", h.getDepartmentByID)
		})
	})

	return r
}
