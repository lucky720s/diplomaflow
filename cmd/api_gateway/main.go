package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/lucky720s/diplomaflow/internal/gateway/config"
	"github.com/lucky720s/diplomaflow/internal/gateway/handler"
	"github.com/lucky720s/diplomaflow/internal/gateway/middleware"
)

func main() {
	cfg := config.Load()
	h, err := handler.NewHandler(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}
	router := gin.Default()
	router.Use(middleware.CorsMiddleware())

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)

		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))

		admin := protected.Group("/admin")
		admin.POST("/universities", h.CreateUniversity)
		admin.GET("/universities", h.ListUniversities)
		admin.POST("/departments", h.CreateDepartment)
		admin.POST("/workflows", h.CreateWorkflow)
		admin.POST("/roles", h.CreateRole)

		projects := protected.Group("/projects")
		projects.POST("", h.CreateProject)
		projects.GET("", h.ListProjects)
		projects.GET("/:id", h.GetProject)
		projects.GET("/student/:student_id", h.GetStudentProjects)

		teams := protected.Group("/teams")
		teams.POST("", h.CreateTeam)
		teams.GET("/:id", h.GetTeam)
		teams.GET("/available-students", h.GetAvailableStudents)
	}

	log.Printf("API Gateway running on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
