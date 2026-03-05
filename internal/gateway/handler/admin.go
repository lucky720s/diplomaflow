package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

func (h *Handler) AdminDeleteUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	requesterID := c.GetInt64("userId")
	requesterRole := c.GetString("role")

	if requesterID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete yourself"})
		return
	}

	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-user-id", strconv.FormatInt(requesterID, 10),
		"x-user-role", requesterRole,
		"x-internal-service", "api_gateway",
	)

	_, err = h.authClient.DeleteUser(ctx, &authv1.DeleteUserRequest{
		UserId:      userID,
		RequesterId: requesterID,
	})
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.NotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		case codes.FailedPrecondition:
			c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": st.Message()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "user deleted successfully",
	})
}

func (h *Handler) GetDepartment(c *gin.Context) {
	depID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || depID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department_id"})
		return
	}

	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-internal-service", "api_gateway",
	)

	resp, err := h.universityClient.GetDepartment(ctx, &universityv1.GetDepartmentRequest{
		DepartmentId: depID,
	})
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.NotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "department not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": st.Message()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"department": mapDepartmentToResponse(resp.Department),
	})
}

func (h *Handler) UpdateDepartment(c *gin.Context) {
	depID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || depID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department_id"})
		return
	}

	var req struct {
		Name         string `json:"name" binding:"required"`
		UniversityID int64  `json:"university_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil { //nolint:govet
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-internal-service", "api_gateway",
	)

	currentDept, err := h.universityClient.GetDepartment(ctx, &universityv1.GetDepartmentRequest{
		DepartmentId: depID,
	})
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.NotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "department not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": st.Message()})
		}
		return
	}

	universityID := currentDept.Department.UniversityId
	if req.UniversityID > 0 {
		universityID = req.UniversityID
	}

	resp, err := h.universityClient.UpdateDepartment(ctx, &universityv1.UpdateDepartmentRequest{
		Department: &universityv1.Department{
			Id:           depID,
			Name:         req.Name,
			UniversityId: universityID,
		},
	})
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.NotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "department not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": st.Message()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"department": mapDepartmentToResponse(resp.Department),
		"message":    "department updated successfully",
	})
}

func (h *Handler) DeleteDepartment(c *gin.Context) {
	depID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || depID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department_id"})
		return
	}

	ctx := metadata.AppendToOutgoingContext(
		c.Request.Context(),
		"x-internal-service", "api_gateway",
	)

	resp, err := h.universityClient.DeleteDepartment(ctx, &universityv1.DeleteDepartmentRequest{
		DepartmentId: depID,
	})
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.NotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "department not found"})
		case codes.FailedPrecondition:
			c.JSON(http.StatusConflict, gin.H{
				"error": "cannot delete department: it has associated users or projects",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": st.Message()})
		}
		return
	}

	if !resp.Success {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete department"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "department deleted successfully",
	})
}

func mapDepartmentToResponse(d *universityv1.Department) map[string]interface{} {
	if d == nil {
		return nil
	}
	return map[string]interface{}{
		"id":            d.Id,
		"name":          d.Name,
		"university_id": d.UniversityId,
	}
}

// GET /api/v1/admin-panel/supervisors/:id/settings
func (h *Handler) GetSupervisorSettings(c *gin.Context) {
	supervisorID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || supervisorID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supervisor_id"})
		return
	}

	departmentID := c.GetInt64("departmentId")

	resp, err := h.adminClient.GetSupervisorSettings(adminPanelCtx(c), &adminv1.GetSupervisorSettingsRequest{
		SupervisorId: supervisorID,
		DepartmentId: departmentID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// PUT /api/v1/admin-panel/supervisors/:id/max-teams
func (h *Handler) UpdateSupervisorMaxTeams(c *gin.Context) {
	supervisorID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || supervisorID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supervisor_id"})
		return
	}

	var req struct {
		MaxTeams int32 `json:"max_teams" binding:"min=0,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { //nolint:govet
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	departmentID := c.GetInt64("departmentId")

	resp, err := h.adminClient.UpdateSupervisorMaxTeams(adminPanelCtx(c), &adminv1.UpdateSupervisorMaxTeamsRequest{
		SupervisorId: supervisorID,
		DepartmentId: departmentID,
		MaxTeams:     req.MaxTeams,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
