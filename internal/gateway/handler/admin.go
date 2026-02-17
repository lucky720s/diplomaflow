package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"google.golang.org/grpc/metadata"
)

func (h *Handler) CreateUniversity(c *gin.Context) {
	var req universityv1.CreateUniversityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := h.universityClient.CreateUniversity(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) ListUniversities(c *gin.Context) {
	res, err := h.universityClient.ListUniversities(c.Request.Context(), &universityv1.ListUniversitiesRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListDepartments(c *gin.Context) {
	uniIDStr := c.Param("id")
	uniID, err := strconv.ParseInt(uniIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid university id"})
		return
	}
	res, err := h.universityClient.ListDepartments(c.Request.Context(), &universityv1.ListDepartmentsRequest{
		UniversityId: uniID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) CreateDepartment(c *gin.Context) {
	var req universityv1.CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := h.universityClient.CreateDepartment(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) CreateWorkflow(c *gin.Context) {
	var req workflowv1.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := h.workflowClient.CreateWorkflow(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

// POST /api/v1/admin/roles
// body: { "slug": "commission", "department_id": 123(optional) }
func (h *Handler) CreateRole(c *gin.Context) {
	var body struct {
		Slug         string `json:"slug" binding:"required"`
		DepartmentID int64  `json:"department_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	departmentID := body.DepartmentID
	if departmentID == 0 {
		departmentID = c.GetInt64("departmentId")
	}

	res, err := h.authClient.CreateDepartmentRole(authIAMCtx(c), &authv1.CreateDepartmentRoleRequest{
		DepartmentId: departmentID,
		Slug:         body.Slug,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) GetUniversity(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid university id"})
		return
	}

	res, err := h.universityClient.GetUniversity(c.Request.Context(), &universityv1.GetUniversityRequest{
		UniversityId: id,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GET /api/v1/roles
func (h *Handler) ListRoles(c *gin.Context) {
	departmentID := c.GetInt64("departmentId")

	res, err := h.authClient.ListDepartmentRoles(authIAMCtx(c), &authv1.ListDepartmentRolesRequest{
		DepartmentId: departmentID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GET /api/v1/roles/:id
func (h *Handler) GetRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}

	res, err := h.authClient.GetDepartmentRole(authIAMCtx(c), &authv1.GetDepartmentRoleRequest{
		RoleId: id,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
func authIAMCtx(c *gin.Context) context.Context {
	ctx := outgoingCtx(c)
	return metadata.AppendToOutgoingContext(ctx, "x-internal-service", "api_gateway")
}
