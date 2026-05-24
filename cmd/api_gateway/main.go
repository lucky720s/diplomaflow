package main

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lucky720s/diplomaflow/internal/gateway"
	"github.com/lucky720s/diplomaflow/internal/gateway/config"
	gatewayhealth "github.com/lucky720s/diplomaflow/internal/gateway/healthz"
	"github.com/lucky720s/diplomaflow/internal/gateway/middleware"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/lucky720s/diplomaflow/pkg/metrics"
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
	registry := metrics.NewRegistry("api_gateway")
	middleware.MustRegisterGatewayMetrics(registry)
	router := gin.New()
	_ = router.SetTrustedProxies([]string{"127.0.0.1"})
	router.Use(gin.Recovery())
	router.Use(middleware.CorsMiddleware(cfg.AllowedOrigins))
	router.Use(middleware.TraceIDMiddleware())
	router.Use(middleware.RequestTimingMiddleware(log.Logger))
	router.Use(middleware.MetricsMiddleware())

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
			auth.POST("/refresh", handler.RefreshToken)
			auth.POST("/logout-cleanup", handler.LogoutCleanup)
		}

		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			authProtected.GET("/sessions", handler.ListSessions)
			authProtected.DELETE("/sessions/:id", handler.RevokeSession)
			authProtected.POST("/logout", handler.Logout)
		}

		users := v1.Group("/users")
		users.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			users.GET("/me", handler.GetMe)
			users.GET("", handler.ListUsers)
			users.GET("/:id", handler.GetUser)
			users.PUT("/:id", handler.UpdateUser)
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
			admin.GET("/departments/:id", handler.GetDepartment)
			admin.PUT("/departments/:id", handler.UpdateDepartment)
			admin.DELETE("/departments/:id", handler.DeleteDepartment)
			admin.POST("/workflows", handler.CreateWorkflow)
			admin.POST("/workflows/:id/states", handler.CreateState)
			admin.POST("/workflows/:id/activate", handler.SetActiveWorkflow)
			admin.POST("/roles", handler.CreateRole)
			admin.POST("/assign-role", handler.AssignRole)
			admin.POST("/users/:id/department-roles", handler.AssignUserDepartmentRole)
			admin.DELETE("/users/:id/department-roles/:role_id", handler.RevokeUserDepartmentRole)
			admin.GET("/users/:id/department-roles", handler.ListUserDepartmentRoles)
			admin.DELETE("/users/:id", handler.AdminDeleteUser)
			admin.POST("/workflows/:id/clone", handler.CloneWorkflow)
			admin.POST("/workflows/:id/version", handler.CreateNewVersion)
			admin.POST("/workflows/:id/validate", handler.ValidateWorkflow)
			admin.POST("/workflows/:id/transitions", handler.CreateTransition)
			admin.PUT("/workflows/:id/transitions/:tid", handler.UpdateTransition)
			admin.DELETE("/workflows/:id/transitions/:tid", handler.DeleteTransition)
			admin.POST("/workflows/:id/states/:sid/actions", handler.CreateStateAction)
		}

		// Admin Panel routes (admin + teacher)
		adminPanel := v1.Group("/admin-panel")
		adminPanel.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		adminPanel.Use(middleware.RBACMiddleware("admin", "teacher"))
		{
			adminPanel.GET("/dashboard", handler.GetAdminDashboard)
			adminPanel.GET("/stats", handler.GetDepartmentStats)
			adminPanel.GET("/students", handler.AdminListStudents)
			adminPanel.GET("/students/:id", handler.AdminGetStudent)
			adminPanel.GET("/teams", handler.AdminListTeams)
			adminPanel.GET("/teams/:id", handler.AdminGetTeamDetails)
			adminPanel.GET("/teams/:id/progress", handler.AdminGetTeamProgress)
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

			// ==================== Pre-Defense ====================
			adminPanel.POST("/pre-defenses/submit", handler.SubmitPreDefenseGW)
			adminPanel.GET("/pre-defenses", handler.ListPreDefenseSubmissionsGW)
			adminPanel.GET("/pre-defenses/schedule", handler.ListScheduledPreDefensesGW)
			adminPanel.GET("/pre-defenses/:id", handler.GetPreDefenseSubmissionGW)
			adminPanel.POST("/pre-defenses/:id/schedule", handler.SchedulePreDefenseGW)
			adminPanel.POST("/pre-defenses/:id/grade", handler.GradePreDefenseGW)
			adminPanel.POST("/pre-defenses/:id/complete", handler.CompletePreDefenseGW)
			adminPanel.POST("/pre-defenses/:id/reschedule", handler.ReschedulePreDefenseGW)
			adminPanel.POST("/pre-defenses/:id/commission", handler.AddPreDefenseCommissionMemberGW)
			adminPanel.DELETE("/pre-defenses/:id/commission/:user_id", handler.RemovePreDefenseCommissionMemberGW)

			adminPanel.GET("/supervisors/:id/settings", handler.GetSupervisorSettings)
			adminPanel.PUT("/supervisors/:id/max-teams", handler.UpdateSupervisorMaxTeams)

			// Workflow review: выставление оценок/допуска преподом
			adminPanel.POST("/projects/:id/states/:state_id/review", handler.SubmitReview)
			adminPanel.GET("/projects/:id/states/:state_id/reviews", handler.GetStateReviews)
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
			projects.GET("/:id/my-grades", handler.GetProjectGradesForStudent)
			projects.GET("/:id/grades", handler.GetProjectGradesForStudent)
			projects.POST("/:id/documents", handler.UploadProjectDocument)
			projects.POST("/:id/submit-document", handler.SubmitProjectDocument)

			// Review (оценка/допуск) по стэйту
			projects.POST("/:id/states/:state_id/review", handler.SubmitReview)
			projects.GET("/:id/states/:state_id/reviews", handler.GetStateReviews)
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
			notifications.POST("/read-all", handler.MarkAllNotificationsRead)
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
			files.DELETE("/:id", handler.DeleteFile)
		}

		forms := v1.Group("/forms")
		forms.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			forms.POST("", handler.SubmitForm)
		}
		topicRegs := v1.Group("/topic-registrations")
		topicRegs.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		topicRegs.Use(middleware.RBACMiddleware("teacher", "admin", "commission"))
		{
			topicRegs.POST("/:id/review", handler.ReviewTopicRegistration)
		}

		// /api/v1/supervisors  (для всех авторизованных)
		supervisors := v1.Group("/supervisors")
		supervisors.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			supervisors.GET("", handler.ListSupervisorsPublic)
		}
		supervisorsPanel := supervisors.Group("")
		supervisorsPanel.Use(middleware.RBACMiddleware("teacher", "admin"))
		{
			supervisorsPanel.GET("/my-requests", handler.ListMySupervisorRequests)
			supervisorsPanel.POST("/requests/:id/respond", handler.RespondToSupervisorRequest)
			supervisorsPanel.POST("/teams/:team_id/claim", handler.TeacherClaimTeam)
			supervisorsPanel.GET("/available-teams", handler.GetAvailableTeams)
			supervisorsPanel.GET("/topic-registrations", handler.ListSupervisorTopicRegistrations)
			supervisorsPanel.GET("/submissions", handler.ListSupervisorSubmissions)
			supervisorsPanel.POST("/topic-registrations/:id/review", handler.ReviewTopicRegistration)
		}

		students := v1.Group("/students")
		students.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			students.GET("/supervisor-requests", handler.ListSupervisorRequestsForStudent)
		}

		norm := v1.Group("/norm-control")
		norm.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		norm.Use(middleware.RBACMiddleware("admin", "norm_control"))
		{
			norm.GET("/pending", handler.NormListPending)
			norm.GET("/documents/:id", handler.NormGetDocument)

			norm.POST("/documents/:id/start", handler.NormStartReview)

			norm.POST("/documents/:id/issues", handler.NormAddIssue)
			norm.PUT("/issues/:id", handler.NormUpdateIssue)
			norm.DELETE("/issues/:id", handler.NormDeleteIssue)

			norm.POST("/documents/:id/approve", handler.NormApprove)
			norm.POST("/documents/:id/return", handler.NormReturn)

			norm.GET("/history/:project_id", handler.NormHistory)
			norm.GET("/statistics", handler.NormStatistics)

			norm.GET("/checklists", handler.NormListChecklists)
			norm.POST("/checklists", handler.NormCreateChecklist)
		}
		adminTech := v1.Group("/admin-tech")
		adminTech.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		adminTech.Use(middleware.RBACMiddleware("admin"))
		{
			// teams
			adminTech.GET("/teams", handler.AdminTechListTeams)
			adminTech.POST("/teams", handler.AdminTechCreateTeam)
			adminTech.GET("/teams/:id", handler.AdminTechGetTeam)
			adminTech.PATCH("/teams/:id", handler.AdminTechUpdateTeam)
			adminTech.DELETE("/teams/:id", handler.AdminTechDeleteTeam)

			// projects
			adminTech.GET("/projects", handler.AdminTechListProjects)
			adminTech.POST("/projects", handler.AdminTechCreateProject)
			adminTech.GET("/projects/:id", handler.AdminTechGetProject)
			adminTech.PATCH("/projects/:id", handler.AdminTechUpdateProject)
			adminTech.POST("/projects/:id/archive", handler.AdminTechArchiveProject)
			adminTech.DELETE("/projects/:id", handler.AdminTechDeleteProject)

			// convenience: projects by team
			adminTech.GET("/teams/:id/projects", handler.AdminTechListTeamProjects)
		}
		departments := v1.Group("/departments")
		departments.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			departments.GET("/:id/dashboard", handler.GetDepartmentDashboard)
			departments.GET("/:id/submissions", handler.GetDepartmentSubmissions)
			departments.GET("/:id/topic-registrations", handler.GetDepartmentTopicRegistrations)
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
		{Name: "admin_norm", Addr: cfg.AdminServiceAddr, ServiceName: "admin.v1.NormControlService"},
	}

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.GET("/readyz", checker.ReadyHandler(targets))

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9080"
	}
	metrics.MustServe(":"+metricsPort, registry)
	log.Info("Gateway metrics endpoint", zap.String("port", metricsPort))
	log.Info("API Gateway starting", zap.String("port", cfg.Port))
	for _, r := range router.Routes() {
		middleware.SetRouteExists(r.Method, r.Path)
	}
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal("failed to start server", zap.Error(err))
	}
}
