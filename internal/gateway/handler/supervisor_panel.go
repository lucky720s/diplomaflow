package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
)

func (h *Handler) ListSupervisorTopicRegistrations(c *gin.Context) {
	supervisorID := c.GetInt64("userId")
	departmentID := c.GetInt64("departmentId")

	page := int32(1)
	pageSize := int32(20)
	if p := c.Query("page"); p != "" {
		if v, _ := strconv.ParseInt(p, 10, 32); v > 0 {
			page = int32(v)
		}
	}

	resp, err := h.adminClient.ListTopicRegistrations(c.Request.Context(), &adminv1.ListTopicRegistrationsRequest{
		DepartmentId: departmentID,
		SupervisorId: supervisorID, // автоматически из JWT
		Status:       c.Query("status"),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListSupervisorSubmissions(c *gin.Context) {
	reviewerID := c.GetInt64("userId")
	departmentID := c.GetInt64("departmentId")

	page := int32(1)
	pageSize := int32(20)
	if p := c.Query("page"); p != "" {
		if v, _ := strconv.ParseInt(p, 10, 32); v > 0 {
			page = int32(v)
		}
	}

	resp, err := h.adminClient.ListSubmissions(c.Request.Context(), &adminv1.ListSubmissionsRequest{
		DepartmentId: departmentID,
		ReviewerId:   reviewerID,
		Status:       c.Query("status"),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
