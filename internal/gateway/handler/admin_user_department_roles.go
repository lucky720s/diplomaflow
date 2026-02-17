package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
)

// POST /api/v1/admin/users/:id/department-roles
// body: { "role_id": 123, "comment": "optional" }
// department_id берём только из JWT (c.GetInt64("departmentId"))
func (h *Handler) AssignUserDepartmentRole(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || targetUserID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var body struct {
		RoleID  int64  `json:"role_id" binding:"required"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.RoleID <= 0 { //nolint:govet
		c.JSON(http.StatusBadRequest, gin.H{"error": "role_id is required"})
		return
	}

	deptID := c.GetInt64("departmentId")
	if deptID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "department_id is required"})
		return
	}

	_, err = h.authClient.AssignDepartmentRole(authIAMCtx(c), &authv1.AssignDepartmentRoleRequest{
		UserId:       targetUserID,
		DepartmentId: deptID,
		RoleId:       body.RoleID,
		Comment:      body.Comment,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DELETE /api/v1/admin/users/:id/department-roles/:role_id
// body optional: { "comment": "optional" }
func (h *Handler) RevokeUserDepartmentRole(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || targetUserID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	roleID, err := strconv.ParseInt(c.Param("role_id"), 10, 64)
	if err != nil || roleID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role_id"})
		return
	}

	var body struct {
		Comment string `json:"comment"`
	}
	_ = c.ShouldBindJSON(&body)

	deptID := c.GetInt64("departmentId")
	if deptID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "department_id is required"})
		return
	}

	_, err = h.authClient.RevokeDepartmentRole(authIAMCtx(c), &authv1.RevokeDepartmentRoleRequest{
		UserId:       targetUserID,
		DepartmentId: deptID,
		RoleId:       roleID,
		Comment:      body.Comment,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GET /api/v1/admin/users/:id/department-roles
// returns { "slugs": ["commission","norm_control"] }
func (h *Handler) ListUserDepartmentRoles(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || targetUserID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	deptID := c.GetInt64("departmentId")
	if deptID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "department_id is required"})
		return
	}

	resp, err := h.authClient.ListUserDepartmentRoles(authIAMCtx(c), &authv1.ListUserDepartmentRolesRequest{
		UserId:       targetUserID,
		DepartmentId: deptID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
