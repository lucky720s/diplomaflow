package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
)

func (h *Handler) GetDepartmentDashboard(c *gin.Context) {
	deptID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	resp, err := h.adminClient.GetDashboard(adminPanelCtx(c), &adminv1.GetDashboardRequest{
		DepartmentId: deptID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
func (h *Handler) GetDepartmentSubmissions(c *gin.Context) {
	deptID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	status := c.Query("status")

	resp, err := h.adminClient.ListSubmissions(adminPanelCtx(c), &adminv1.ListSubmissionsRequest{
		DepartmentId: deptID,
		Status:       status,
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
func (h *Handler) GetDepartmentTopicRegistrations(c *gin.Context) {
	deptID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	status := c.Query("status")

	resp, err := h.adminClient.ListTopicRegistrations(adminPanelCtx(c), &adminv1.ListTopicRegistrationsRequest{
		DepartmentId: deptID,
		Status:       status,
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
