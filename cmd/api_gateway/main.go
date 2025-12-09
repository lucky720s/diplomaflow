package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lucky720s/diplomaflow/internal/gateway"
	"github.com/lucky720s/diplomaflow/internal/gateway/config"
	"github.com/lucky720s/diplomaflow/internal/gateway/middleware"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	var cfg config.Config
	if err := config.Load("config.yaml", &cfg); err != nil {
		panic("failed to load config: " + err.Error())
	}

	log := logger.New(cfg.Env)
	defer log.Sync()

	h, cleanup, err := gateway.InitializeApp(&cfg, log)
	if err != nil {
		log.Fatal("Failed to initialize handlers", zap.Error(err))
	}
	defer cleanup()

	if cfg.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())

	router.Use(func(c *gin.Context) {
		traceID := c.Request.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = "gen-" + time.Now().Format("20060102150405")
		}
		c.Set("trace_id", traceID)
		c.Header("X-Trace-ID", traceID)
		c.Next()
	})
	router.Use(middleware.CorsMiddleware())

	v1 := router.Group("/api/v1")
	{

		auth := v1.Group("/auth")
		{
			auth.POST("/register", h.Register)
			auth.POST("/login", h.Login)
			//auth.POST("/validate", h.ValidateToken)
		}
		public := v1.Group("/public")
		{
			public.GET("/universities", h.ListUniversities)
			public.GET("/universities/:id/departments", h.ListDepartments)
		}
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))

		//protected.GET("/users", h.ListUsers)

		projects := protected.Group("/projects")
		{
			projects.POST("", h.CreateProject)
			projects.GET("/:id", h.GetProjectDetails)
			projects.GET("", h.ListProjects)
			// projects.PUT("/:id", h.UpdateProject)
			// projects.DELETE("/:id", h.DeleteProject)
		}
		protected.GET("/students/:student_id/projects", h.GetStudentProjects)
		teams := protected.Group("/teams")
		{
			teams.POST("", h.CreateTeam)
			teams.GET("/:id", h.GetTeam)
			teams.GET("/available-students", h.GetAvailableStudents)

			teams.PUT("/:id/assign-project", h.AssignProjectToTeam)
			teams.GET("/invites", h.GetMyInvites)
			teams.POST("/invites/:id/respond", h.RespondToInvite)
		}
		universities := protected.Group("/universities")
		{
			universities.POST("", h.CreateUniversity)
			//universities.GET("/:id", h.GetUniversity)
			//universities.GET("", h.ListUniversities)
		}

		roles := protected.Group("/roles")
		{
			roles.POST("", h.CreateRole)
			//roles.GET("", h.ListRoles)
			//roles.GET("/:id", h.GetRole)
		}

		workflows := protected.Group("/workflows")
		{
			workflows.POST("", h.CreateWorkflow)
			//workflows.GET("/:id", h.GetWorkflow)
			// workflows.PUT("/:id/status", h.UpdateWorkflowStatus)
		}
		notifications := protected.Group("/notifications")
		{
			notifications.GET("", h.ListNotifications)
			notifications.PUT("/:id/read", h.MarkNotificationRead)
		}
		files := protected.Group("/files")
		{
			files.POST("/upload", h.UploadFile)
			files.GET("/:id", h.DownloadFile)
		}
		forms := protected.Group("/forms")
		{
			forms.POST("/submit", h.SubmitForm)
		}
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("API Gateway running", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("listen error", zap.Error(err))
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down API Gateway...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exited")
}
