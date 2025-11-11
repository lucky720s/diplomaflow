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
	auth_pb "github.com/lucky720s/diplomaflow/pkg/protobuf/auth"
	project_pb "github.com/lucky720s/diplomaflow/pkg/protobuf/project"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ApiGateWay struct {
	authClient    auth_pb.AuthServiceClient
	projectClient project_pb.ProjectServiceClient
}

func main() {
	authConn, err := grpc.Dial("auth_service:8082", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect to auth service: %v", err)
	}
	projectConn, err := grpc.Dial("project_service:8083", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect to project service: %v", err)
	}
	gateWay := &ApiGateWay{
		authClient:    auth_pb.NewAuthServiceClient(authConn),
		projectClient: project_pb.NewProjectServiceClient(projectConn),
	}
	router := gin.Default()
	authRoutes := router.Group("/api/v1/auth")
	{
		authRoutes.POST("/register", gateWay.Register)
		authRoutes.POST("/login", gateWay.login)
	}
	api := router.Group("/api/v1")
	api.Use(AuthMiddleware())
	{
		api.POST("/projects", gateWay.CreateProject)
		api.GET("/projects", gateWay.ListProjects)
		api.GET("/projects/:id", gateWay.GetProject)
		api.PUT("/projects/:id", gateWay.UpdateProject)
		api.DELETE("/projects/:id", gateWay.DeleteProject)
	}
	err = router.Run(":8080")
	if err != nil {
		return
	}
}

func (g *ApiGateWay) Register(c *gin.Context) {
	var req auth_pb.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Println("Connecting to auth_service gRPC...")
	res, err := g.authClient.Register(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (g *ApiGateWay) login(c *gin.Context) {
	var req auth_pb.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.authClient.Login(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (g *ApiGateWay) CreateProject(c *gin.Context) {
	var req project_pb.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.projectClient.CreateProject(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
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
		c.Set("userId", claims["sub"])

		c.Next()
	}
}

func (g *ApiGateWay) GetProject(c *gin.Context) {
	id := c.Param("id")
	projectID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := g.projectClient.GetProject(context.Background(),
		&project_pb.GetProjectRequest{ProjectId: int64(projectID)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
func (g *ApiGateWay) ListProjects(c *gin.Context) {
	res, err := g.projectClient.ListProjects(context.Background(), &project_pb.ListProjectsRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
func (g *ApiGateWay) UpdateProject(c *gin.Context) {
	userIDClaim, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := int64(userIDClaim.(float64))
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
	req := project_pb.UpdateProjectRequest{
		ProjectId: int64(projectID),
		Topic:     reqBody.Topic,
		UserId:    userID,
	}
	res, err := g.projectClient.UpdateProject(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (g *ApiGateWay) DeleteProject(c *gin.Context) {
	userIDClaim, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := int64(userIDClaim.(float64))
	id := c.Param("id")
	projectID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req := &project_pb.DeleteProjectRequest{
		ProjectId: int64(projectID),
		UserId:    userID,
	}
	_, err = g.projectClient.DeleteProject(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, gin.H{})
}
