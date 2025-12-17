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
	"google.golang.org/protobuf/types/known/structpb"
)

type UserInfo struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}
type EnrichedProjectResponse struct {
	ID             int64      `json:"id"`
	Title          string     `json:"title"`
	WorkflowID     int64      `json:"workflow_id"`
	CurrentStateID int64      `json:"current_state_id"`
	CurrentState   *StateInfo `json:"current_state"`
	Status         string     `json:"status"`
}

type StateInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
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
	authAddr := os.Getenv("AUTH_SERVICE_ADDR")
	projectAddr := os.Getenv("PROJECT_SERVICE_ADDR")
	universityAddr := os.Getenv("UNIVERSITY_SERVICE_ADDR")
	teamAddr := os.Getenv("TEAM_SERVICE_ADDR")
	workflowAddr := os.Getenv("WORKFLOW_SERVICE_ADDR")
	roleAddr := os.Getenv("ROLE_SERVICE_ADDR")

	if authAddr == "" {
		authAddr = "auth_service:8082"
	}
	if projectAddr == "" {
		projectAddr = "project_service:8083"
	}
	if universityAddr == "" {
		universityAddr = "university_service:8081"
	}
	if teamAddr == "" {
		teamAddr = "team_service:8084"
	}
	if workflowAddr == "" {
		workflowAddr = "workflow_service:8085"
	}
	if roleAddr == "" {
		roleAddr = "role_service:8086"
	}

	authConn, _ := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	projectConn, _ := grpc.Dial(projectAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	universityConn, _ := grpc.Dial(universityAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	teamConn, _ := grpc.Dial(teamAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	workflowConn, _ := grpc.Dial(workflowAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	roleConn, _ := grpc.Dial(roleAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))

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
		admin.Use(AuthMiddleware())
		admin.POST("/universities", gateway.CreateUniversity)
		admin.GET("/universities", gateway.ListUniversities)
		admin.GET("/universities/:id", gateway.GetUniversity)
		admin.PATCH("/universities/:id", gateway.UpdateUniversity)
		admin.DELETE("/universities/:id", gateway.DeleteUniversity)
		admin.POST("/departments", gateway.CreateDepartment)
		admin.GET("/universities/:id/departments", gateway.ListDepartments)
		admin.GET("/departments/:id", gateway.GetDepartment)
		admin.PATCH("/departments/:id", gateway.UpdateDepartment)
		admin.DELETE("/departments/:id", gateway.DeleteDepartment)

		admin.POST("/workflows", gateway.CreateWorkflow)
		admin.POST("/states", gateway.CreateState)
		admin.POST("/roles", gateway.CreateRole)
		admin.POST("/assign-role", gateway.AssignRole)

		projects := v1.Group("/projects")
		projects.Use(AuthMiddleware())
		projects.POST("", gateway.CreateProject)
		projects.GET("", gateway.ListProjects)
		projects.GET("/:id", gateway.GetProject)
		projects.POST("/:id/actions", gateway.PerformProjectAction)

		teams := v1.Group("/teams")
		teams.Use(AuthMiddleware())
		teams.POST("", gateway.CreateTeam)
		teams.GET("", gateway.ListTeams)
		teams.GET("/:id", gateway.GetTeam)
		teams.PATCH("/:id", gateway.UpdateTeam)
		teams.DELETE("/:id", gateway.DeleteTeam)
		teams.GET("/available-users", gateway.ListAvailableUsers)
		teams.POST("/:id/members", gateway.AddMember)
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

func (g *ApiGateWay) GetUniversity(c *gin.Context) {
	id := c.Param("id")
	universityID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.universityClient.GetUniversity(context.Background(), &universityv1.GetUniversityRequest{
		UniversityId: universityID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
func (g *ApiGateWay) ListUniversities(c *gin.Context) {
	res, err := g.universityClient.ListUniversities(context.Background(), &universityv1.ListUniversitiesRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
func (g *ApiGateWay) UpdateUniversity(c *gin.Context) {
	id := c.Param("id")
	universityID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var reqBody struct {
		Name      string `json:"name"`
		ShortName string `json:"short_name"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req := universityv1.UpdateUniversityRequest{
		University: &universityv1.University{
			Id:        universityID,
			Name:      reqBody.Name,
			ShortName: reqBody.ShortName},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"name", "short_name"}}}
	res, err := g.universityClient.UpdateUniversity(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
func (g *ApiGateWay) DeleteUniversity(c *gin.Context) {
	id := c.Param("id")
	universityID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err = g.universityClient.DeleteUniversity(context.Background(), &universityv1.DeleteUniversityRequest{
		UniversityId: universityID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
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

func (g *ApiGateWay) GetDepartment(c *gin.Context) {
	id := c.Param("id")
	departmentID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.universityClient.GetDepartment(context.Background(), &universityv1.GetDepartmentRequest{
		DepartmentId: departmentID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
func (g *ApiGateWay) ListDepartments(c *gin.Context) {
	id := c.Param("id")
	universityID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return

	}
	res, err := g.universityClient.ListDepartments(context.Background(), &universityv1.ListDepartmentsRequest{
		UniversityId: universityID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
func (g *ApiGateWay) UpdateDepartment(c *gin.Context) {
	id := c.Param("id")
	departmentID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var reqBody struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req := universityv1.UpdateDepartmentRequest{
		Department: &universityv1.Department{
			Id:   departmentID,
			Name: reqBody.Name,
		},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"name"},
		},
	}
	res, err := g.universityClient.UpdateDepartment(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (g *ApiGateWay) DeleteDepartment(c *gin.Context) {
	id := c.Param("id")
	departmentID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err = g.universityClient.DeleteDepartment(context.Background(), &universityv1.DeleteDepartmentRequest{
		DepartmentId: departmentID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
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
	var reqBody struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	departmentID := c.MustGet("departmentId").(int64)
	req := &teamv1.CreateTeamRequest{
		Name:         reqBody.Name,
		DepartmentId: departmentID,
	}
	ctx := newContextWithAuth(c)
	res, err := g.teamClient.CreateTeam(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}
func (g *ApiGateWay) ListTeams(c *gin.Context) {
	ctx := newContextWithAuth(c)
	res, err := g.teamClient.ListTeams(ctx, &teamv1.ListTeamsRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	c.JSON(http.StatusOK, res)
}
func (g *ApiGateWay) GetTeam(c *gin.Context) {
	id := c.Param("id")
	teamID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := newContextWithAuth(c)
	res, err := g.teamClient.GetTeam(ctx, &teamv1.GetTeamRequest{TeamId: teamID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (g *ApiGateWay) UpdateTeam(c *gin.Context) {
	id := c.Param("id")
	teamID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var reqBody struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req := &teamv1.UpdateTeamRequest{
		Team: &teamv1.Team{
			Id: teamID, Name: reqBody.Name},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"name"}},
	}
	ctx := newContextWithAuth(c)
	res, err := g.teamClient.UpdateTeam(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (g *ApiGateWay) DeleteTeam(c *gin.Context) {
	id := c.Param("id")
	teamID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req := &teamv1.DeleteTeamRequest{
		TeamId: teamID}
	ctx := newContextWithAuth(c)
	_, err = g.teamClient.DeleteTeam(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
func (g *ApiGateWay) AddMember(c *gin.Context) {
	id := c.Param("id")
	teamID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var reqBody struct {
		UserID int64 `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req := &teamv1.AddMemberRequest{
		TeamId: teamID,
		UserId: reqBody.UserID}
	ctx := newContextWithAuth(c)
	_, err = g.teamClient.AddMember(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
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

func (g *ApiGateWay) ListAvailableUsers(c *gin.Context) {
	id := c.Param("did")
	departmentId, err := strconv.ParseInt(id, 10, 64)
	if departmentId == 0 {
		return
	}
	res, err := g.teamClient.ListAvailableUsers(c, &teamv1.ListAvailableUsersRequest{DepartmentId: departmentId})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
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

func (g *ApiGateWay) CreateState(c *gin.Context) {
	var req workflowv1.CreateStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.workflowClient.CreateState(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (g *ApiGateWay) CreateTransition(c *gin.Context) {
	var req workflowv1.CreateTransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.workflowClient.CreateTransition(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (g *ApiGateWay) CreateStateAction(c *gin.Context) {
	var req workflowv1.CreateStateActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.workflowClient.CreateStateAction(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (g *ApiGateWay) CreateProject(c *gin.Context) {
	var req projectv1.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := newContextWithAuth(c)
	res, err := g.projectClient.CreateProject(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (g *ApiGateWay) PerformProjectAction(c *gin.Context) {
	id := c.Param("id")
	projectID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	var reqBody struct {
		Action  string                 `json:"action"`
		Payload map[string]interface{} `json:"payload"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payloadStruct, err := structpb.NewStruct(reqBody.Payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload format"})
		return
	}

	req := &projectv1.PerformStateActionRequest{
		ProjectId: projectID,
		Action:    reqBody.Action,
		Payload:   payloadStruct,
	}

	ctx := newContextWithAuth(c)
	_, err = g.projectClient.PerformStateAction(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "action processed"})
}

func (g *ApiGateWay) GetProject(c *gin.Context) {
	id := c.Param("id")
	projectID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}
	ctx := newContextWithAuth(c)

	projectRes, err := g.projectClient.GetProject(ctx, &projectv1.GetProjectRequest{ProjectId: projectID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var stateInfo *StateInfo
	stateRes, err := g.workflowClient.GetState(context.Background(), &workflowv1.GetStateRequest{StateId: projectRes.GetCurrentStateId()})
	if err == nil && stateRes != nil {
		stateInfo = &StateInfo{
			Name:        stateRes.GetName(),
			Description: stateRes.GetDescription(),
		}
	}

	finalResponse := EnrichedProjectResponse{
		ID:             projectRes.GetId(),
		Title:          projectRes.GetTitle(),
		WorkflowID:     projectRes.GetWorkflowId(),
		CurrentStateID: projectRes.GetCurrentStateId(),
		CurrentState:   stateInfo,
		Status:         projectRes.GetStatus(),
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
