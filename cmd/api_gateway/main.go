package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lucky720s/diplomaflow/internal/gateway"
	"github.com/lucky720s/diplomaflow/internal/gateway/config"
	gatewayhealth "github.com/lucky720s/diplomaflow/internal/gateway/healthz"
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

	log := logger.New(cfg.Env)
	defer log.Sync()

	// FAIL-FAST: JWT secret must be set via env in real deployments
	// В config.yaml он может быть пустым, но в ENV он должен прийти обязательно.
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required (gateway validates JWT locally)")
	}

	handler, cleanup, err := gateway.InitializeApp(&cfg, log)
	if err != nil {
		log.Fatal("failed to initialize gateway", zap.Error(err))
	}
	defer cleanup()

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CorsMiddleware(cfg.AllowedOrigins))
	router.Use(middleware.TraceIDMiddleware())
	router.Use(middleware.RequestTimingMiddleware(log.Logger))

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
			admin.POST("/users/:id/department-roles", handler.AssignUserDepartmentRole)
			admin.DELETE("/users/:id/department-roles/:role_id", handler.RevokeUserDepartmentRole)
			admin.GET("/users/:id/department-roles", handler.ListUserDepartmentRoles)
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
			projects.GET("", handler.ListProjects)
			projects.GET("/:id", handler.GetProject)
			projects.GET("/:id/details", handler.GetProjectDetails)
			projects.POST("/:id/actions", handler.PerformProjectAction)

			// project-first flows тоже лучше закрыть admin-only, иначе можно обойти team-first
			projects.POST("/:id/topic-registration", handler.SubmitTopicRegistration)
			projects.POST("/:id/supervisor-request", handler.CreateSupervisorRequest)
			projects.DELETE("/:id/supervisor-requests/:request_id", handler.CancelSupervisorRequest)
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

			teams.POST("/:id/members", handler.AddMember)
			teams.DELETE("/:id/members/:user_id", handler.RemoveMember)

			teams.POST("/:id/leave", handler.LeaveTeam)
			teams.POST("/:id/transfer-leadership", handler.TransferLeadership)

			teams.POST("/join-by-code",
				middleware.RateLimitByUserMiddleware(rdb, 10, time.Hour),
				handler.JoinTeamByCode,
			)

			teams.POST("/:id/invite-code/regenerate", handler.RegenerateInviteCode)

			// NEW: team-first supervisor request (до создания проекта)
			teams.POST("/:id/supervisor-request", handler.CreateSupervisorRequestByTeam)
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

			// NEW: справочные конфиги кафедры/команды из workflow_service
			workflows.GET("/department-config", handler.GetDepartmentWorkflowConfig)
			workflows.GET("/team-configuration", handler.GetTeamConfiguration)
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
			notifications.DELETE("/:id", handler.DeleteNotification)
		}

		// NEW: onboarding status endpoint (BFF для фронта)
		onboarding := v1.Group("/onboarding")
		onboarding.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			onboarding.GET("", handler.GetOnboardingStatus)
		}

		tasks := v1.Group("/tasks")
		tasks.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			// My tasks
			tasks.GET("/my", handler.GetMyTasks)
			tasks.GET("/overdue", handler.GetOverdueTasks)
			tasks.GET("/upcoming", handler.GetUpcomingDeadlines)

			// Task CRUD
			tasks.POST("", handler.CreateTask)
			tasks.GET("", handler.ListTasks)
			tasks.GET("/:id", handler.GetTask)
			tasks.PATCH("/:id", handler.UpdateTask)
			tasks.DELETE("/:id", handler.DeleteTask)

			// Task operations
			tasks.POST("/:id/move", handler.MoveTask)
			tasks.POST("/:id/assign", handler.AssignTask)
			tasks.DELETE("/:id/assign", handler.UnassignTask)

			// Comments
			tasks.POST("/:id/comments", handler.CreateTaskComment)
			tasks.GET("/:id/comments", handler.ListTaskComments)
			tasks.DELETE("/:id/comments/:comment_id", handler.DeleteTaskComment)

			// Activity
			tasks.GET("/:id/activity", handler.GetTaskActivity)

			// Watchers
			tasks.POST("/:id/watchers", handler.AddTaskWatcher)
			tasks.DELETE("/:id/watchers", handler.RemoveTaskWatcher)
			tasks.GET("/:id/watchers", handler.ListTaskWatchers)
		}

		boards := v1.Group("/boards")
		boards.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			boards.GET("/my", handler.ListMyBoards)
			boards.GET("/project/:project_id", handler.GetBoardByProject)

			boards.GET("/:board_id", handler.GetBoard)
			boards.PATCH("/:board_id", handler.UpdateBoard)
			boards.GET("/:board_id/stats", handler.GetBoardStats)

			boards.GET("/:board_id/columns", handler.ListColumns)
			boards.POST("/:board_id/columns", handler.CreateColumn)
			boards.POST("/:board_id/columns/reorder", handler.ReorderColumns)
			boards.PATCH("/:board_id/columns/:column_id", handler.UpdateColumn)
			boards.DELETE("/:board_id/columns/:column_id", handler.DeleteColumn)
			boards.POST("/:board_id/columns/:column_id/reorder-tasks", handler.ReorderTasks)
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
			supervisors.POST("/teams/:team_id/claim", handler.TeacherClaimTeam)
			supervisors.GET("/available-teams", handler.GetAvailableTeams)
		}
	}

	checker := gatewayhealth.NewChecker(rdb, 2*time.Second)
	defer checker.Close()

	targets := []gatewayhealth.ServiceTarget{
		{Name: "auth", Addr: cfg.AuthServiceAddr, ServiceName: "auth.v1.AuthService"},
		{Name: "project", Addr: cfg.ProjectServiceAddr, ServiceName: "project.v1.ProjectService"},
		{Name: "task", Addr: cfg.TaskServiceAddr, ServiceName: "task.v1.TaskService"},
		{Name: "team", Addr: cfg.TeamServiceAddr, ServiceName: "team.v1.TeamService"},
		{Name: "university", Addr: cfg.UniversityServiceAddr, ServiceName: "university.v1.UniversityService"},
		{Name: "workflow", Addr: cfg.WorkflowServiceAddr, ServiceName: "workflow.v1.WorkflowService"},
		{Name: "notification", Addr: cfg.NotificationServiceAddr, ServiceName: "notification.v1.NotificationService"},
		{Name: "file", Addr: cfg.FileServiceAddr, ServiceName: "file.v1.FileService"},
		{Name: "form", Addr: cfg.FormServiceAddr, ServiceName: "form.v1.FormService"},
		{Name: "admin", Addr: cfg.AdminServiceAddr, ServiceName: "admin.v1.AdminService"},
	}

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.GET("/readyz", checker.ReadyHandler(targets))

	log.Info("API Gateway starting", zap.String("port", cfg.Port))
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal("failed to start server", zap.Error(err))
	}
}
