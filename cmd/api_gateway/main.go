package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type UserInfo struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}
type EnrichedProjectResponse struct {
	ID           int64     `json:"id"`
	Topic        string    `json:"topic"`
	Supervisor   *UserInfo `json:"supervisor"`
	TeamID       int64     `json:"team_id,omitempty"`
	DepartmentID int64     `json:"department_id"`
	Status       string    `json:"status"`
}

type ApiGateWay struct {
	authClient       authv1.AuthServiceClient
	projectClient    projectv1.ProjectServiceClient
	universityClient universityv1.UniversityServiceClient
	teamClient       teamv1.TeamServiceClient
	workflowClient   workflowv1.WorkflowServiceClient
	roleClient       rolev1.RoleServiceClient
}

func main() {
	authConn, _ := grpc.Dial("auth_service:8082", grpc.WithTransportCredentials(insecure.NewCredentials()))
	projectConn, _ := grpc.Dial("project_service:8083", grpc.WithTransportCredentials(insecure.NewCredentials()))
	universityConn, _ := grpc.Dial("university_service:8081", grpc.WithTransportCredentials(insecure.NewCredentials()))
	teamConn, _ := grpc.Dial("team_service:8084", grpc.WithTransportCredentials(insecure.NewCredentials()))
	workflowConn, _ := grpc.Dial("workflow_service:8085", grpc.WithTransportCredentials(insecure.NewCredentials()))
	roleConn, _ := grpc.Dial("role_service:8086", grpc.WithTransportCredentials(insecure.NewCredentials()))
	gateway := &ApiGateWay{
		authClient:       authv1.NewAuthServiceClient(authConn),
		projectClient:    projectv1.NewProjectServiceClient(projectConn),
		universityClient: universityv1.NewUniversityServiceClient(universityConn),
		teamClient:       teamv1.NewTeamServiceClient(teamConn),
		workflowClient:   workflowv1.NewWorkflowServiceClient(workflowConn),
		roleClient:       rolev1.NewRoleServiceClient(roleConn),
	}
	router := gin.Default()
	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		auth.POST("/register", gateway.Register)
		auth.POST("/login", gateway.Login)

		admin := v1.Group("/admin")
		//admin.Use(AuthMiddleware())
		admin.POST("/universities", gateway.CreateUniversity)
		admin.POST("/departments", gateway.CreateDepartment)
		admin.POST("/workflows", gateway.CreateWorkflow)
		admin.POST("/stages", gateway.CreateStage)
		admin.POST("/roles", gateway.CreateRole)
		admin.POST("/assign-role", gateway.AssignRole)

		projects := v1.Group("/projects")
		//projects.Use(AuthMiddleware())
		projects.POST("", gateway.ProposeProject)
		projects.GET("", gateway.ListProjects)
		projects.GET("/:id", gateway.GetProject)
		projects.PATCH("/:id", gateway.UpdateProject)
		projects.DELETE("/:id", gateway.DeleteProject)
		projects.POST("/:id/accept-supervision", gateway.AcceptSupervision)
		projects.POST("/:id/advance-stage", gateway.AdvanceProjectStage)

		teams := v1.Group("/teams")
		//teams.Use(AuthMiddleware())
		teams.POST("", gateway.CreateTeam)
		teams.DELETE("/:id/members", gateway.RemoveMember)
	}
	log.Println("API GateWay is listening on :8080")
	err := router.Run(":8080")
	if err != nil {
		return
	}
}

func (g *ApiGateWay) Register(c *gin.Context) {
	var req authv1.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.authClient.Register(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (g *ApiGateWay) Login(c *gin.Context) {
	var req authv1.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.authClient.Login(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (g *ApiGateWay) CreateUniversity(c *gin.Context) {
	var req universityv1.CreateUniversityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.universityClient.CreateUniversity(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (g *ApiGateWay) CreateDepartment(c *gin.Context) {
	var req universityv1.CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.universityClient.CreateDepartment(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}
func (g *ApiGateWay) CreateWorkflow(c *gin.Context) {
	var req workflowv1.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.workflowClient.CreateWorkflow(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (g *ApiGateWay) CreateStage(c *gin.Context) {
	var req workflowv1.CreateStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	res, err := g.workflowClient.CreateStage(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (g *ApiGateWay) CreateRole(c *gin.Context) {
	var req rolev1.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := newContextWithAuth(c)
	res, err := g.roleClient.CreateRole(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (g *ApiGateWay) AssignRole(c *gin.Context) {
	var req authv1.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.authClient.AssignRole(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)

}

func (g *ApiGateWay) CreateTeam(c *gin.Context) {
	var req teamv1.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := newContextWithAuth(c)
	res, err := g.teamClient.CreateTeam(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (g *ApiGateWay) RemoveMember(c *gin.Context) {
	var req teamv1.RemoveMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := newContextWithAuth(c)
	_, err := g.teamClient.RemoveMember(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
func (g *ApiGateWay) ProposeProject(c *gin.Context) {
	var req projectv1.ProposeProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := newContextWithAuth(c)
	res, err := g.projectClient.ProposeProject(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}
func (g *ApiGateWay) AcceptSupervision(c *gin.Context) {
	id := c.Param("id")
	projectID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req := &projectv1.AcceptSupervisionRequest{ProjectId: int64(projectID)}
	ctx := newContextWithAuth(c)
	res, err := g.projectClient.AcceptSupervision(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
func (g *ApiGateWay) GetProject(c *gin.Context) {
	id := c.Param("id")
	projectID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := newContextWithAuth(c)
	projectRes, err := g.projectClient.GetProject(ctx, &projectv1.GetProjectRequest{ProjectId: int64(projectID)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	supervisorInfo, _ := g.authClient.GetUser(context.Background(),
		&authv1.GetUserRequest{UserId: projectRes.Project.GetSupervisorId()})
	stageInfo, _ := g.workflowClient.GetStage(context.Background(),
		&workflowv1.GetStageRequest{StageId: projectRes.Project.GetCurrentStageId()})
	status := "UNKNOWN"
	if stageInfo != nil {
		status = stageInfo.Stage.GetName()
	}
	if projectRes.Project.GetCompletedAt() != nil {
		status = "COMPLETED"
	}
	finalResponse := EnrichedProjectResponse{
		ID:           projectRes.Project.GetId(),
		Topic:        projectRes.Project.GetTopic(),
		Supervisor:   &UserInfo{ID: supervisorInfo.GetId(), Email: supervisorInfo.GetEmail()},
		TeamID:       projectRes.Project.GetTeamId(),
		DepartmentID: projectRes.Project.GetDepartmentId(),
		Status:       status,
	}
	c.JSON(http.StatusOK, finalResponse)
}

func (g *ApiGateWay) ListProjects(c *gin.Context) {
	ctx := newContextWithAuth(c)
	res, err := g.projectClient.ListProjects(ctx, &projectv1.ListProjectsRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
func (g *ApiGateWay) UpdateProject(c *gin.Context) {
	id := c.Param("id")
	projectID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var reqBody struct {
		Topic string `json:"topic"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req := &projectv1.UpdateProjectRequest{
		Project: &projectv1.Project{
			Id:    int64(projectID),
			Topic: reqBody.Topic},
		UpdateMask: &field_mask.FieldMask{Paths: []string{"topic"}},
	}
	ctx := newContextWithAuth(c)
	res, err := g.projectClient.UpdateProject(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}
func (g *ApiGateWay) DeleteProject(c *gin.Context) {
	id := c.Param("id")
	projectID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req := &projectv1.DeleteProjectRequest{
		ProjectId: int64(projectID),
	}
	ctx := newContextWithAuth(c)
	_, err = g.projectClient.DeleteProject(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (g *ApiGateWay) AdvanceProjectStage(c *gin.Context) {
	id := c.Param("id")
	projectID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req := &projectv1.AdvanceProjectStageRequest{
		ProjectId: projectID,
	}
	ctx := newContextWithAuth(c)
	res, err := g.projectClient.AdvanceProjectStage(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No Authorization header"})
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header"})
			return
		}
		tokenString := parts[1]
		jwtSecret := os.Getenv("JWT_SECRET")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}
		c.Set("userId", int64(claims["sub"].(float64)))
		c.Set("departmentId", int64(claims["did"].(float64)))
		c.Set("universityId", int64(claims["uid"].(float64)))
		if roles, ok := claims["roles"].([]interface{}); ok {
			var roleStrings []string
			for _, role := range roles {
				roleStrings = append(roleStrings, strconv.FormatInt(int64(role.(float64)), 10))
			}
			c.Set("userRoles", roleStrings)
		}
		c.Next()
	}
}
func newContextWithAuth(c *gin.Context) context.Context {
	md := metadata.New(map[string]string{
		"user-id":       strconv.FormatInt(c.MustGet("userId").(int64), 10),
		"department-id": strconv.FormatInt(c.MustGet("departmentId").(int64), 10),
		"university-id": strconv.FormatInt(c.MustGet("universityId").(int64), 10),
	})
	if roles, exists := c.Get("userRoles"); exists {
		if roleStrings, ok := roles.([]string); ok {
			md.Set("user-roles",
				roleStrings...)
		}
	}
	return metadata.NewOutgoingContext(context.Background(), md)
}
