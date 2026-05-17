package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	"github.com/lucky720s/diplomaflow/internal/workflow/engine"
	"github.com/lucky720s/diplomaflow/internal/workflow/outboxpoller"
	"github.com/lucky720s/diplomaflow/internal/workflow/plugins/builtin"
	"github.com/lucky720s/diplomaflow/internal/workflow/postcommit"
	"github.com/lucky720s/diplomaflow/internal/workflow/runtimegrpc"
	"github.com/lucky720s/diplomaflow/pkg/config"
	grpcpkg "github.com/lucky720s/diplomaflow/pkg/grpc"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/lucky720s/diplomaflow/pkg/metrics"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(grpcpkg.DefaultRetryServiceConfig()),
		grpc.WithUnaryInterceptor(grpcpkg.TimeoutInterceptor(10*time.Second)),
	)
}
func ensureDefaultOnEnterNotifications(db *gorm.DB, log *zap.Logger) {
	var states []workflow.State
	if err := db.Find(&states).Error; err != nil {
		log.Warn("default notifications: failed to list states", zap.Error(err))
		return
	}

	type tpl struct{ title, msg, link, nType string }
	byType := map[string]tpl{
		"TEAM_FORMATION": {
			title: "Сформируйте команду",
			msg:   "Этап «{{state_name}}»: добавьте участников и отправьте приглашения.",
			link:  "/team",
			nType: "workflow_team_formation_enter",
		},
		"SUPERVISOR_SELECTION": {
			title: "Выберите научного руководителя",
			msg:   "Этап «{{state_name}}»: выберите научного руководителя для проекта.",
			link:  "/diplom/select-supervisor",
			nType: "workflow_supervisor_selection_enter",
		},
		"TOPIC_APPROVAL": {
			title: "Отправьте тему на согласование",
			msg:   "Этап «{{state_name}}»: заполните и отправьте тему диплома.",
			link:  "/diplom",
			nType: "workflow_topic_required",
		},
		"DOCUMENT_UPLOAD": {
			title: "Нужно загрузить документы",
			msg:   "Этап «{{state_name}}»: загрузите необходимые файлы.",
			link:  "/diplom",
			nType: "workflow_document_upload_required",
		},
		"FORM_SUBMIT": {
			title: "Нужно заполнить форму",
			msg:   "Этап «{{state_name}}»: заполните и отправьте форму.",
			link:  "/diplom",
			nType: "workflow_form_required",
		},
		"DEFENSE": {
			title: "Подготовка к защите",
			msg:   "Этап «{{state_name}}»: проверьте требования и готовьтесь к защите.",
			link:  "/diplom",
			nType: "workflow_defense_stage_enter",
		},
		"COMPLETED": {
			title: "Проект завершён",
			msg:   "Workflow проекта завершён.",
			link:  "/diplom",
			nType: "workflow_completed",
		},
	}

	for _, st := range states {
		t, ok := byType[strings.ToUpper(strings.TrimSpace(st.Type))]
		if !ok {
			continue
		}

		// если уже есть хотя бы один SEND_NOTIFICATION на ON_ENTER — ничего не добавляем
		var cnt int64
		if err := db.Model(&workflow.StateAction{}).
			Where("state_id = ? AND type = ? AND trigger = ? AND is_enabled = ?", st.ID, "SEND_NOTIFICATION", "ON_ENTER", true).
			Count(&cnt).Error; err != nil {
			log.Warn("default notifications: count actions failed", zap.Int64("state_id", st.ID), zap.Error(err))
			continue
		}
		if cnt > 0 {
			continue
		}

		cfg := map[string]interface{}{
			"title":      t.title,
			"message":    t.msg,
			"link":       t.link,
			"type":       t.nType,
			"recipients": "team",
		}
		b, _ := json.Marshal(cfg)

		a := &workflow.StateAction{
			StateID:    st.ID,
			Name:       "AUTO_NOTIFY_ON_ENTER",
			Type:       "SEND_NOTIFICATION",
			Trigger:    "ON_ENTER",
			OrderIndex: 0,
			Config:     datatypes.JSON(b),
			IsEnabled:  true,
			MaxRetries: 3,
		}

		if err := db.Create(a).Error; err != nil {
			log.Warn("default notifications: create action failed", zap.Int64("state_id", st.ID), zap.Error(err))
			continue
		}

		log.Info("default notifications: created ON_ENTER SEND_NOTIFICATION",
			zap.Int64("state_id", st.ID),
			zap.String("state_type", st.Type),
		)
	}
}

func main() {
	var cfg workflow.Config
	if err := config.Load("config.yaml", &cfg); err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	log := logger.New(cfg.Env)
	defer func() { _ = log.Sync() }()

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect db", zap.Error(err))
	}
	ensureDefaultOnEnterNotifications(db, log.Logger)

	// Connect notification
	notifConn, err := dial(cfg.Services.NotificationAddr)
	if err != nil {
		log.Fatal("failed to connect notification service", zap.String("addr", cfg.Services.NotificationAddr), zap.Error(err))
	}
	defer notifConn.Close()
	notifClient := notificationv1.NewNotificationServiceClient(notifConn)

	// Connect project
	projectConn, err := dial(cfg.Services.ProjectAddr)
	if err != nil {
		log.Fatal("failed to connect project service", zap.String("addr", cfg.Services.ProjectAddr), zap.Error(err))
	}
	defer projectConn.Close()
	projectClient := projectv1.NewProjectServiceClient(projectConn)

	// Connect team
	teamConn, err := dial(cfg.Services.TeamAddr)
	if err != nil {
		log.Fatal("failed to connect team service", zap.String("addr", cfg.Services.TeamAddr), zap.Error(err))
	}
	defer teamConn.Close()
	teamClient := teamv1.NewTeamServiceClient(teamConn)

	repo := workflow.NewRepository(db)
	svc := workflow.NewService(repo, log.Logger)
	base := workflow.NewHandler(svc, log.Logger)

	reviewRepo := workflow.NewReviewRepository(db)
	reviewSvc := workflow.NewReviewService(reviewRepo, repo, log.Logger)

	// Plugins
	builtin.RegisterAll(notifClient, teamClient, reviewSvc)
	log.Info("Builtin plugins registered", zap.Strings("plugins", builtin.RegisteredPlugins()))

	eng := engine.NewWorkflowEngine(repo, teamClient, log.Logger)
	h := runtimegrpc.New(base, eng, reviewSvc, projectClient, log.Logger)

	pcWorker := postcommit.NewWorker(db, projectClient, teamClient, log.Logger)
	pcGRPC := postcommit.NewGRPCServer(pcWorker, log.Logger)

	poller, err := outboxpoller.New(db, pcWorker, log.Logger, outboxpoller.Config{
		Enabled:           cfg.OutboxPoller.Enabled,
		Topic:             cfg.OutboxPoller.Topic,
		IntervalSeconds:   cfg.OutboxPoller.IntervalSeconds,
		BatchSize:         cfg.OutboxPoller.BatchSize,
		Table:             cfg.OutboxPoller.Table,
		IDColumn:          cfg.OutboxPoller.IDColumn,
		TopicColumn:       cfg.OutboxPoller.TopicColumn,
		StatusColumn:      cfg.OutboxPoller.StatusColumn,
		EventTypeColumn:   cfg.OutboxPoller.EventTypeColumn,
		PayloadColumn:     cfg.OutboxPoller.PayloadColumn,
		ProcessedAtColumn: cfg.OutboxPoller.ProcessedAtColumn,
		PendingStatus:     cfg.OutboxPoller.PendingStatus,
		ProcessingStatus:  cfg.OutboxPoller.ProcessingStatus,
		ProcessedStatus:   cfg.OutboxPoller.ProcessedStatus,
	})
	if err != nil {
		log.Fatal("failed to init outbox poller", zap.Error(err))
	}

	pollerCtx, pollerCancel := context.WithCancel(context.Background())
	go poller.Start(pollerCtx)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	reg := metrics.NewRegistry("workflow_service")
	reg.MustRegister(metrics.GRPCCollectors()...)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(metrics.UnaryServerInterceptor("workflow_service")),
	)
	workflowv1.RegisterWorkflowServiceServer(grpcServer, h)
	workflowv1.RegisterWorkflowActionsServiceServer(grpcServer, pcGRPC)

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9085"
	}
	metrics.MustServe(":"+metricsPort, reg)
	log.Info("Workflow metrics endpoint", zap.String("port", metricsPort))

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("workflow.v1.WorkflowService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("workflow.v1.WorkflowActionsService", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	go func() {
		log.Info("Workflow Service starting", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info("Shutdown signal received")
	pollerCancel()

	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	healthServer.SetServingStatus("workflow.v1.WorkflowService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	healthServer.SetServingStatus("workflow.v1.WorkflowActionsService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	time.Sleep(300 * time.Millisecond)
	grpcServer.GracefulStop()
	log.Info("Workflow Service stopped")
}
