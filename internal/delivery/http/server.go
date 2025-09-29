package http

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/internal/middleware"
)

// InitRoutes настраивает маршруты и middleware
func (h *Handler) InitRoutes() *chi.Mux {
	r := chi.NewRouter()

	// --- Стандартные Middleware ---
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// --- API v1 ---
	r.Route("/api/v1", func(r chi.Router) {
		// TODO: Здесь должны быть публичные роуты, например, логин
		// r.Post("/login", h.loginHandler)

		// --- Группа защищенных роутов ---
		r.Group(func(r chi.Router) {
			// Применяем middleware для проверки токена ко всем роутам в этой группе
			r.Use(middleware.AuthMiddleware)

			// --- Студенты ---
			r.Route("/students", func(r chi.Router) {
				// Создавать может только студент (согласно вашей таблице)
				r.With(middleware.RoleMiddleware(domain.RoleStudent)).Post("/", h.registerStudent)
				// Просматривать могут все авторизованные пользователи
				r.Get("/", h.listStudents)
				r.Get("/{id}", h.getStudentByID)
				// Удалять могут только админы
				r.With(middleware.RoleMiddleware(domain.RoleSysAdmin, domain.RoleDeptAdmin)).Delete("/{id}", h.deleteStudent)
			})

			// --- Кафедры ---
			r.Route("/departments", func(r chi.Router) {
				// Создавать могут только админы
				r.With(middleware.RoleMiddleware(domain.RoleSysAdmin, domain.RoleDeptAdmin)).Post("/", h.createDepartment)
				// Просматривать могут все авторизованные
				r.Get("/{id}", h.getDepartmentByID)
			})
		})
	})

	return r
}
