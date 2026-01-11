package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lucky720s/diplomaflow/internal/gateway"
	"github.com/lucky720s/diplomaflow/internal/gateway/config"
	"github.com/lucky720s/diplomaflow/internal/gateway/middleware"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	var cfg config.Config
	if err := config.Load("config.yaml", &cfg); err != nil {
		panic(err)
	}
	//test
	log := logger.New(cfg.Env)
	defer log.Sync()

	handler, cleanup, err := gateway.InitializeApp(&cfg, log)
	if err != nil {
		log.Fatal("failed to initialize gateway", zap.Error(err))
	}
	defer cleanup()

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	router := gin.Default()
	router.Use(middleware.CorsMiddleware(cfg.AllowedOrigins))

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
			auth.POST("/refresh", handler.RefreshToken)
			auth.POST("/logout", handler.Logout)
		}
		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			authProtected.GET("/sessions", handler.ListSessions)
			authProtected.DELETE("/sessions/:id", handler.RevokeSession)
		}
		users := v1.Group("/users")
		users.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			users.GET("/me", handler.GetMe)
			users.GET("", handler.ListUsers)
		}
		universities := v1.Group("/universities")
		{
			universities.GET("", handler.ListUniversities)
			universities.GET("/:id", handler.GetUniversity)
			universities.GET("/:id/departments", handler.ListDepartments)
		}

		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		admin.Use(middleware.RBACMiddleware("admin"))
		{
			admin.POST("/universities", handler.CreateUniversity)
			admin.POST("/departments", handler.CreateDepartment)
			admin.POST("/workflows", handler.CreateWorkflow)
			admin.POST("/workflows/:id/states", handler.CreateState)
			admin.POST("/workflows/:id/activate", handler.SetActiveWorkflow)
			admin.POST("/roles", handler.CreateRole)
			admin.POST("/assign-role", handler.AssignRole)
			admin.POST("/workflows/:id/clone", handler.CloneWorkflow)
			admin.POST("/workflows/:id/version", handler.CreateNewVersion)
			admin.POST("/workflows/:id/validate", handler.ValidateWorkflow)
			admin.POST("/workflows/:id/transitions", handler.CreateTransition)
			admin.PUT("/workflows/:id/transitions/:tid", handler.UpdateTransition)
			admin.DELETE("/workflows/:id/transitions/:tid", handler.DeleteTransition)
			admin.POST("/workflows/:id/states/:sid/actions", handler.CreateStateAction)
		}

		// Admin Panel routes (Commission & Tech Support)
		adminPanel := v1.Group("/admin-panel")
		adminPanel.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		adminPanel.Use(middleware.RBACMiddleware("admin", "commission", "tech_support"))
		{
			adminPanel.GET("/dashboard", handler.GetAdminDashboard)
			adminPanel.GET("/stats", handler.GetDepartmentStats)
			adminPanel.GET("/students", handler.AdminListStudents)
			adminPanel.GET("/students/:id", handler.AdminGetStudent)
			adminPanel.GET("/teams", handler.AdminListTeams)
			adminPanel.GET("/teams/:id", handler.AdminGetTeamDetails)
			adminPanel.PATCH("/teams/:id", handler.AdminUpdateTeam)
			adminPanel.DELETE("/teams/:id", handler.AdminDeleteTeam)
			adminPanel.GET("/supervisors", handler.ListSupervisors)
			adminPanel.POST("/supervisors/assign", handler.AssignSupervisor)
			adminPanel.GET("/submissions", handler.ListSubmissions)
			adminPanel.GET("/submissions/:id", handler.GetSubmission)
			adminPanel.POST("/submissions/:id/review", handler.ReviewSubmission)
			adminPanel.GET("/projects/:id/grades", handler.GetProjectGrades)
			adminPanel.POST("/projects/:id/grades", handler.SetStepGrade)
			adminPanel.GET("/workflow/progress", handler.GetWorkflowProgress)
			adminPanel.GET("/pending-reviews", handler.ListPendingReviews)
			adminPanel.GET("/topic-registrations", handler.ListTopicRegistrations)
			adminPanel.GET("/topic-registrations/:id", handler.GetTopicRegistration)
			adminPanel.POST("/topic-registrations/:id/review", handler.ReviewTopicRegistration)
			adminPanel.GET("/supervisor-requests", handler.ListAllSupervisorRequests)
			adminPanel.GET("/supervisor-requests/:id", handler.GetSupervisorRequestDetails)
			adminPanel.POST("/supervisor-requests/:id/respond", handler.RespondToSupervisorRequest)
		}

		projects := v1.Group("/projects")
		projects.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			projects.POST("", handler.CreateProject)
			projects.GET("", handler.ListProjects)
			projects.GET("/:id", handler.GetProject)
			projects.GET("/:id/details", handler.GetProjectDetails)
			projects.POST("/:id/actions", handler.PerformProjectAction)
		}

		teams := v1.Group("/teams")
		teams.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			teams.POST("", handler.CreateTeam)
			teams.GET("", handler.ListTeams)
			teams.GET("/:id", handler.GetTeam)
			teams.PATCH("/:id", handler.UpdateTeam)
			teams.DELETE("/:id", handler.DeleteTeam)
			teams.GET("/my", handler.GetMyTeam)
			teams.GET("/available-students", handler.GetAvailableStudents)
			teams.POST("/:id/assign-project", handler.AssignProjectToTeam)
			teams.POST("/:id/members", handler.AddMember)
			teams.DELETE("/:id/members", handler.RemoveMember)
			teams.POST("/:id/topic-registration", handler.SubmitTopicRegistration)
			teams.POST("/:id/supervisor-request", handler.CreateSupervisorRequest)
			teams.DELETE("/supervisor-requests/:id", handler.CancelSupervisorRequest)
		}
		invites := v1.Group("/invites")
		invites.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			invites.GET("", handler.GetMyInvites)
			invites.POST("/:id/respond", handler.RespondToInvite)
		}

		workflows := v1.Group("/workflows")
		workflows.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			workflows.GET("", handler.ListWorkflows)
			workflows.GET("/:id", handler.GetWorkflow)
			workflows.GET("/:id/full", handler.GetWorkflowFull)
			workflows.GET("/:id/states", handler.ListStates)
			workflows.GET("/:id/transitions", handler.ListTransitions)
			workflows.GET("/transitions/available", handler.GetAvailableTransitions)
			workflows.GET("/states/:state_id/config", handler.GetStepConfiguration)
		}

		roles := v1.Group("/roles")
		roles.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			roles.GET("", handler.ListRoles)
			roles.GET("/:id", handler.GetRole)
		}

		notifications := v1.Group("/notifications")
		notifications.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			notifications.GET("", handler.ListNotifications)
			notifications.POST("/:id/read", handler.MarkNotificationRead)
		}

		files := v1.Group("/files")
		files.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		files.Use(middleware.RateLimitMiddleware(rdb, 10, time.Minute))
		{
			files.POST("/upload", handler.UploadFile)
			files.GET("/:id", handler.DownloadFile)
		}

		forms := v1.Group("/forms")
		forms.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			forms.POST("", handler.SubmitForm)
		}

		supervisors := v1.Group("/supervisors")
		supervisors.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		supervisors.Use(middleware.RBACMiddleware("teacher", "admin"))
		{
			supervisors.GET("/my-requests", handler.ListMySupervisorRequests)
			supervisors.POST("/requests/:id/respond", handler.RespondToSupervisorRequest)
		}
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Info("API Gateway starting", zap.String("port", cfg.Port))
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal("failed to start server", zap.Error(err))
	}
}
