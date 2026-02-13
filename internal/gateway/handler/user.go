package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
)

func (h *Handler) ListUsers(c *gin.Context) {
	universityID := c.GetInt64("universityId")
	departmentID := c.GetInt64("departmentId")

	// allow explicit override (useful for admin panels)
	if v := c.Query("department_id"); v != "" {
		if did, err := strconv.ParseInt(v, 10, 64); err == nil && did > 0 {
			departmentID = did
		}
	}
	if v := c.Query("university_id"); v != "" {
		if uid, err := strconv.ParseInt(v, 10, 64); err == nil && uid > 0 {
			universityID = uid
		}
	}

	role := c.Query("role")

	page := int32(1)
	pageSize := int32(20)

	if p := c.Query("page"); p != "" {
		if v, err := strconv.ParseInt(p, 10, 32); err == nil && v > 0 {
			page = int32(v)
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.ParseInt(ps, 10, 32); err == nil && v > 0 {
			pageSize = int32(v)
		}
	}

	res, err := h.authClient.ListUsers(c.Request.Context(), &authv1.ListUsersRequest{
		UniversityId: universityID,
		DepartmentId: departmentID, // ✅ IMPORTANT
		Role:         role,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}
