package project

import (
	"context"
	"errors"
	"strconv"

	authv1 "github.com/lucky720s/diplomaflow/protobuf/auth/v1"
	projectv1 "github.com/lucky720s/diplomaflow/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/protobuf/team/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/protobuf/workflow/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type Handler struct {
	projectv1.UnimplementedProjectServiceServer
	repo           Repository
	authClient     authv1.AuthServiceClient
	teamClient     teamv1.TeamServiceClient
	workflowClient workflowv1.WorkflowServiceClient
}

func NewHandler(repo Repository, authClient authv1.AuthServiceClient, teamClient teamv1.TeamServiceClient, workflowClient workflowv1.WorkflowServiceClient) *Handler {
	return &Handler{
		repo:           repo,
		authClient:     authClient,
		teamClient:     teamClient,
		workflowClient: workflowClient,
	}
}
func getAuthInfoFromContext(ctx context.Context) (userID, departmentID, universityID int64, roles []int64, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		err = status.Errorf(codes.FailedPrecondition, "metadata.FromIncomingContext failed")
		return
	}
	parse := func(key string) (int64, error) {
		values := md.Get(key)
		if len(values) == 0 {
			return 0, status.Errorf(codes.Unauthenticated, "%s is not metadata", key)
		}
		return strconv.ParseInt(values[0], 10, 64)
	}
	if userID, err = parse("user-id"); err != nil {
		return
	}
	if departmentID, err = parse("department-id"); err != nil {
		return
	}
	if universityID, err = parse("university-id"); err != nil {
		return
	}
	roleValues := md.Get("user-roles")
	if len(roleValues) > 0 {
		for _, r := range roleValues {
			roleInt, _ := strconv.ParseInt(r, 10, 64)
			roles = append(roles, roleInt)
		}
	}
	return
}

func toProto(p *Project) *projectv1.Project {
	var completedAt *timestamppb.Timestamp
	if p.CompletedAt != nil {
		completedAt = timestamppb.New(*p.CompletedAt)
	}
	return &projectv1.Project{
		Id:             p.ID,
		Topic:          p.Topic,
		SupervisorId:   p.SupervisorID,
		DepartmentId:   p.DepartmentID,
		TeamId:         p.TeamID,
		CurrentStageId: p.CurrentStageID,
		CompletedAt:    completedAt,
	}
}
func (h *Handler) ProposeProject(ctx context.Context, req *projectv1.ProposeProjectRequest) (*projectv1.ProposeProjectResponse, error) {
	_, departmentID, _, _, err := getAuthInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}
	userRes, err := h.authClient.GetUser(ctx, &authv1.GetUserRequest{
		UserId: req.GetSupervisorId()})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "supervisor with id %d does not exist: %v", req.GetSupervisorId(), err)
	}
	if userRes.GetDepartmentId() != departmentID {
		return nil, status.Errorf(codes.PermissionDenied, "supervisor does not belong to this department")
	}
	if req.GetTeamId() != 0 {
		teamRes, err := h.teamClient.GetTeam(ctx, &teamv1.GetTeamRequest{
			TeamId: req.GetTeamId()})
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "team with id %d does not exist: %v", req.GetTeamId(), err)
		}
		if teamRes.Team.GetDepartmentId() != departmentID {
			return nil, status.Errorf(codes.PermissionDenied, "team does not belong to this department")
		}

	}
	workflowRes, err := h.workflowClient.GetWorkflow(ctx, &workflowv1.GetWorkflowRequest{
		DepartmentId: departmentID})
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to get workflow for department: %v", err)
	}
	if len(workflowRes.GetStages()) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "no stages")
	}
	firstStageID := workflowRes.Stages[0].Id
	project := &Project{
		Topic:          req.GetTopic(),
		SupervisorID:   req.GetSupervisorId(),
		DepartmentID:   departmentID,
		TeamID:         req.GetTeamId(),
		CurrentStageID: firstStageID,
	}
	createdProject, err := h.repo.CreateProject(ctx, project)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to create project: %v", err)
	}
	return &projectv1.ProposeProjectResponse{Project: toProto(createdProject)}, nil
}

func (h *Handler) AcceptSupervision(ctx context.Context, req *projectv1.AcceptSupervisionRequest) (*projectv1.AcceptSupervisionResponse, error) {
	userID, departmentID, _, _, err := getAuthInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}
	project, err := h.repo.GetProjectByID(ctx, req.GetProjectId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "project not found")
	}
	if project.DepartmentID != departmentID {
		return nil, status.Error(codes.PermissionDenied, "department does not belong to this department")
	}
	if project.SupervisorID != userID {
		return nil, status.Error(codes.PermissionDenied, "user does not belong to this supervisor")
	}
	nextStageRes, err := h.workflowClient.GetNextStage(ctx, &workflowv1.GetNextStageRequest{
		CurrentStageId: project.CurrentStageID})
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to get next stage for project: %v", err)
	}
	if nextStageRes.GetNextStageId() == 0 {
		return nil, status.Error(codes.FailedPrecondition, "no next stage for project")
	}
	if err := h.repo.UpdateProjectStage(ctx, project.ID, nextStageRes.GetNextStageId()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update next stage for project: %v", err)
	}
	project.CurrentStageID = nextStageRes.GetNextStageId()
	return &projectv1.AcceptSupervisionResponse{Project: toProto(project)}, nil
}
func (h *Handler) GetProject(ctx context.Context, req *projectv1.GetProjectRequest) (*projectv1.GetProjectResponse, error) {
	_, departmentID, _, _, err := getAuthInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}
	project, err := h.repo.GetProjectByID(ctx, req.GetProjectId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "project not found")
		}
		return nil, status.Error(codes.Internal, "could not get project")
	}
	if project.DepartmentID != departmentID {
		return nil, status.Errorf(codes.PermissionDenied, "department does not belong to this department")
	}
	return &projectv1.GetProjectResponse{Project: toProto(project)}, nil
}
func (h *Handler) ListProjects(ctx context.Context, req *projectv1.ListProjectsRequest) (*projectv1.ListProjectsResponse, error) {
	_, departmentID, _, _, err := getAuthInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}
	projects, err := h.repo.ListProjects(ctx, departmentID)
	var resProjects []*projectv1.Project
	for _, project := range projects {
		resProjects = append(resProjects, toProto(project))
	}
	return &projectv1.ListProjectsResponse{Projects: resProjects}, nil
}

func (h *Handler) UpdateProject(ctx context.Context, req *projectv1.UpdateProjectRequest) (*projectv1.UpdateProjectResponse, error) {
	userID, departmentID, _, _, err := getAuthInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}
	project, err := h.repo.GetProjectByID(ctx, req.GetProject().GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "project not found")
	}
	if project.DepartmentID != departmentID {
		return nil, status.Errorf(codes.PermissionDenied, "department does not belong to this department")
	}
	if project.SupervisorID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "supervisor does not belong to this user")
	}
	updatedProject, err := h.repo.UpdateProject(ctx, req.GetProject(), req.GetUpdateMask())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update project: %v", err)
	}
	return &projectv1.UpdateProjectResponse{Project: toProto(updatedProject)}, nil
}

func (h *Handler) DeleteProject(ctx context.Context, req *projectv1.DeleteProjectRequest) (*projectv1.DeleteProjectResponse, error) {
	userID, departmentID, _, _, err := getAuthInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}
	project, err := h.repo.GetProjectByID(ctx, req.GetProjectId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "project not found")
	}
	if project.DepartmentID != departmentID {
		return nil, status.Errorf(codes.PermissionDenied, "department does not belong to this department")
	}
	if project.SupervisorID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "supervisor does not belong to this user")
	}
	err = h.repo.DeleteProject(ctx, req.GetProjectId())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete project")
	}
	return &projectv1.DeleteProjectResponse{Success: true}, nil
}
func (h *Handler) AdvanceProjectStage(ctx context.Context, req *projectv1.AdvanceProjectStageRequest) (*projectv1.AdvanceProjectStageResponse, error) {
	_, departmentID, _, userRoles, err := getAuthInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}
	project, err := h.repo.GetProjectByID(ctx, req.GetProjectId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "project not found")
	}
	if project.DepartmentID != departmentID {
		return nil, status.Errorf(codes.PermissionDenied, "department does not belong to this department")
	}
	currentStageRes, err := h.workflowClient.GetStage(ctx, &workflowv1.GetStageRequest{StageId: project.CurrentStageID})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get current stage for project")
	}
	isAuthorized := false
	for _, userRoleID := range userRoles {
		if userRoleID == currentStageRes.Stage.GetResponsibleRoleId() {
			isAuthorized = true
			break
		}
	}
	if !isAuthorized {
		return nil, status.Errorf(codes.PermissionDenied, "not authorized")
	}
	nextStageRes, err := h.workflowClient.GetNextStage(ctx, &workflowv1.GetNextStageRequest{CurrentStageId: project.CurrentStageID})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get next stage for project")
	}
	if nextStageRes.GetNextStageId() == 0 {
		if err := h.repo.CompleteProject(ctx, req.GetProjectId()); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to complete project: %v", err)
			project, _ = h.repo.GetProjectByID(ctx, req.GetProjectId())
		}
		return &projectv1.AdvanceProjectStageResponse{Project: toProto(project)}, nil
	}
	if err := h.repo.UpdateProjectStage(ctx, req.GetProjectId(), nextStageRes.GetNextStageId()); err != nil {
		return nil, status.Error(codes.Internal, "failed to update project stage")
	}
	project.CurrentStageID = nextStageRes.GetNextStageId()
	return &projectv1.AdvanceProjectStageResponse{Project: toProto(project)}, nil
}
