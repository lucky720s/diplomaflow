package main

import (
	"context"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lucky720s/diplomaflow/internal/gateway"
	"github.com/lucky720s/diplomaflow/internal/gateway/config"
	gatewayhealth "github.com/lucky720s/diplomaflow/internal/gateway/healthz"
	"github.com/lucky720s/diplomaflow/internal/gateway/middleware"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/lucky720s/diplomaflow/pkg/metrics"
	"github.com/lucky720s/diplomaflow/pkg/realtime"
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

	// Realtime: подписываемся на Redis Pub/Sub и фаним события в WS-клиентов.
	rtCtx, rtCancel := context.WithCancel(context.Background())
	defer rtCancel()
	rtSub := realtime.NewSubscriber(rdb)
	defer rtSub.Close()
	go func() {
		for ev := range rtSub.Events(rtCtx) {
			handler.DispatchRealtime(ev)
		}
	}()
	registry := metrics.NewRegistry("api_gateway")
	middleware.MustRegisterGatewayMetrics(registry)
	router := gin.New()
	_ = router.SetTrustedProxies([]string{"127.0.0.1"})
	// Single upload size limit (aligned with nginx + file service).
	maxUpload := cfg.MaxUploadBytesOrDefault()
	router.MaxMultipartMemory = maxUpload
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
			admin.POST("/workflows", handler.AdminCreateWorkflow)
			admin.GET("/workflows", handler.AdminListWorkflows)
			admin.GET("/workflows/active", handler.AdminGetActiveWorkflowByDepartment)
			admin.GET("/workflows/:id", handler.AdminGetWorkflow)
			admin.GET("/workflows/:id/full", handler.AdminGetWorkflowFull)
			admin.PATCH("/workflows/:id", handler.AdminUpdateWorkflow)
			admin.DELETE("/workflows/:id", handler.AdminDeleteWorkflow)
			admin.POST("/workflows/:id/clone", handler.AdminCloneWorkflow)
			admin.POST("/workflows/:id/activate", handler.AdminActivateWorkflow)
			admin.POST("/workflows/:id/version", handler.AdminCreateNewVersion)
			admin.POST("/workflows/:id/validate", handler.AdminValidateWorkflow)

			// States
			admin.GET("/workflows/:id/states", handler.AdminListStates)
			admin.POST("/workflows/:id/states", handler.AdminCreateState)
			admin.PATCH("/workflows/:id/states/:sid", handler.AdminUpdateState)
			admin.DELETE("/workflows/:id/states/:sid", handler.AdminDeleteState)
			admin.POST("/workflows/:id/states/reorder", handler.AdminReorderStates)
			admin.POST("/workflows/:id/states/:sid/actions", handler.CreateStateAction)

			// Transitions
			admin.GET("/workflows/:id/transitions", handler.AdminListTransitions)
			admin.POST("/workflows/:id/transitions", handler.AdminCreateTransition)
			admin.PATCH("/workflows/:id/transitions/:tid", handler.AdminUpdateTransition)
			admin.DELETE("/workflows/:id/transitions/:tid", handler.AdminDeleteTransition)

			// Project workflow runtime
			admin.GET("/workflow-projects", handler.AdminListWorkflowProjects)
			admin.GET("/workflow-dashboard", handler.AdminGetWorkflowDashboardStats)
			admin.GET("/projects/:id/workflow", handler.AdminGetProjectWorkflowState)
			admin.GET("/projects/:id/workflow/history", handler.AdminListProjectWorkflowHistory)
			admin.POST("/projects/:id/workflow/advance", handler.AdminAdvanceProjectWorkflow)
			admin.POST("/projects/:id/workflow/rollback", handler.AdminRollbackProjectWorkflow)
			admin.POST("/projects/:id/workflow/set-state", handler.AdminSetProjectWorkflowState)

			admin.POST("/roles", handler.CreateRole)
			admin.POST("/assign-role", handler.AssignRole)
			admin.POST("/users/:id/department-roles", handler.AssignUserDepartmentRole)
			admin.DELETE("/users/:id/department-roles/:role_id", handler.RevokeUserDepartmentRole)
			admin.GET("/users/:id/department-roles", handler.ListUserDepartmentRoles)
			admin.DELETE("/users/:id", handler.AdminDeleteUser)
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
			// Pre-defense grading is a commission action; finalization is a
			// dean_office action. RBAC above admits admin/teacher; these
			// per-route guards narrow it to the correct department role
			// (admin passes as an explicit, audited override). A plain teacher
			// or a supervisor without the dept role gets 403.
			adminPanel.POST("/pre-defenses/:id/grade",
				middleware.RequireDepartmentRole("commission"), handler.GradePreDefenseGW)
			adminPanel.POST("/pre-defenses/:id/complete",
				middleware.RequireDepartmentRole("dean_office"), handler.CompletePreDefenseGW)
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
			projects.POST("/:id/documents", middleware.MaxBodySize(maxUpload), handler.UploadProjectDocument)
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

		// Mobile BFF: агрегированные эндпоинты под Flutter-приложение.
		mobile := v1.Group("/mobile")
		mobile.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			mobile.GET("/home", handler.GetMobileHome)

			// Push (FCM) device tokens.
			mobile.GET("/devices", handler.ListDevices)
			mobile.POST("/devices", handler.RegisterDevice)
			mobile.DELETE("/devices", handler.UnregisterDevice)
		}

		// Chat: история через REST + realtime через WebSocket.
		chat := v1.Group("/chat")
		chat.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			chat.GET("/conversations", handler.ListConversations)
			chat.POST("/conversations", handler.CreateConversation)
			chat.GET("/conversations/:id", handler.GetConversation)
			chat.GET("/conversations/:id/messages", handler.ListMessages)
			chat.POST("/conversations/:id/messages", handler.SendMessage)
			chat.POST("/conversations/:id/read", handler.MarkChatRead)

			// WebSocket: realtime-канал (Authorization: Bearer <token>).
			chat.GET("/ws", handler.ChatWebSocket)
		}

		// Единый realtime-канал (доска, уведомления) поверх Redis Pub/Sub.
		realtimeGroup := v1.Group("/realtime")
		realtimeGroup.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			realtimeGroup.GET("/ws", handler.RealtimeWebSocket)
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

			// Attachments
			tasks.GET("/:id/attachments", handler.ListTaskAttachments)
			tasks.POST("/:id/attachments", handler.AddTaskAttachment)
			tasks.DELETE("/:id/attachments/:attachment_id", handler.RemoveTaskAttachment)

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
			files.POST("/upload", middleware.MaxBodySize(maxUpload), handler.UploadFile)
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

			// Supervisor Projects (supervisor-facing, owner-scoped via JWT).
			supervisorsPanel.GET("/projects", handler.ListSupervisorProjects)
			supervisorsPanel.GET("/projects/:project_id", handler.GetSupervisorProjectDetails)
			supervisorsPanel.GET("/projects/:project_id/submissions", handler.ListSupervisorProjectSubmissions)
			supervisorsPanel.GET("/projects/:project_id/submissions/:submission_id", handler.GetSupervisorProjectSubmission)
			supervisorsPanel.GET("/projects/:project_id/grades", handler.GetSupervisorProjectGrades)
			supervisorsPanel.GET("/projects/:project_id/grading-history", handler.GetSupervisorProjectGradingHistory)
			supervisorsPanel.GET("/projects/:project_id/workflow-history", handler.GetSupervisorProjectWorkflowHistory)
			supervisorsPanel.GET("/projects/:project_id/files", handler.ListSupervisorProjectFiles)
			supervisorsPanel.GET("/projects/:project_id/timeline", handler.GetSupervisorProjectTimeline)
			supervisorsPanel.POST("/projects/:project_id/submissions/:submission_id/review", handler.ReviewSupervisorProjectSubmission)
			supervisorsPanel.POST("/projects/:project_id/states/:state_id/review", handler.SubmitSupervisorStateReview)
		}

		students := v1.Group("/students")
		students.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			students.GET("/supervisor-requests", handler.ListSupervisorRequestsForStudent)
		}
		//norm-control
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

		// Commission: pre-defense review (grading). Only the `commission`
		// department role (admin override allowed). Commission grades but does
		// NOT finalize the workflow — that is dean_office's job.
		commission := v1.Group("/commission")
		commission.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		commission.Use(middleware.RequireDepartmentRole("commission"))
		{
			commission.GET("/pre-defenses", handler.ListPreDefenseSubmissionsGW)
			commission.GET("/pre-defenses/:id", handler.GetPreDefenseSubmissionGW)
			commission.POST("/pre-defenses/:id/grade", handler.GradePreDefenseGW)
		}

		// Dean Office: final administrative decision on pre-defense. Only the
		// `dean_office` department role (admin override allowed). Finalization
		// runs the workflow transition centrally via the admin/workflow service.
		deanOffice := v1.Group("/dean-office")
		deanOffice.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		deanOffice.Use(middleware.RequireDepartmentRole("dean_office"))
		{
			deanOffice.GET("/pre-defenses", handler.ListPreDefenseSubmissionsGW)
			deanOffice.GET("/pre-defenses/:id", handler.GetPreDefenseSubmissionGW)
			deanOffice.POST("/pre-defenses/:id/complete", handler.CompletePreDefenseGW)
		}

		//antiplagiat
		antiplag := v1.Group("/antiplagiat")
		antiplag.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		antiplag.Use(middleware.RBACMiddleware("admin", "antiplagiat"))
		{
			antiplag.GET("/pending", handler.AntiplagListPending)
			antiplag.GET("/documents/:id", handler.AntiplagGetDocument)

			antiplag.POST("/documents/:id/start", handler.AntiplagStartReview)
			antiplag.POST("/documents/:id/scores", handler.AntiplagSetScores)

			antiplag.POST("/documents/:id/comments", handler.AntiplagAddComment)
			antiplag.PUT("/comments/:id", handler.AntiplagUpdateComment)
			antiplag.DELETE("/comments/:id", handler.AntiplagDeleteComment)

			antiplag.POST("/documents/:id/approve", handler.AntiplagApprove)
			antiplag.POST("/documents/:id/return", handler.AntiplagReturn)

			antiplag.GET("/history/:project_id", handler.AntiplagHistory)
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

		// Teacher Department Progress (read-only, department-scoped).
		//
		// NOT the admin panel. Access is NOT a base-role check: it requires the
		// dynamic department role `department_progress_viewer` (roles /
		// user_role_assignments), verified per-request and per-department inside
		// each handler via requireDepartmentProgressAccess. Therefore only
		// AuthMiddleware is applied here — no RBACMiddleware("teacher").
		//
		// GetDepartmentProgressTeamDetails already returns workflow, history and
		// grades, so no separate workflow/history/grades routes are exposed yet.
		teacherDP := v1.Group("/teacher/department-progress")
		teacherDP.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			teacherDP.GET("/summary", handler.TeacherDPSummary)
			teacherDP.GET("/teams", handler.TeacherDPListTeams)
			teacherDP.GET("/teams/:team_id", handler.TeacherDPTeamDetails)
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
		{Name: "chat", Addr: cfg.ChatServiceAddr, ServiceName: "chat.v1.ChatService"},
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
