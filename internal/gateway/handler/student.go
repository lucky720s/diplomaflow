package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
)

func (h *Handler) ListSupervisorRequestsForStudent(c *gin.Context) {
	supervisorID, _ := strconv.ParseInt(c.Query("supervisor_id"), 10, 64)

	ctx := adminPanelCtx(c)

	resp, err := h.adminClient.ListSupervisorRequests(ctx, &adminv1.ListSupervisorRequestsReq{
		SupervisorId: supervisorID,
		DepartmentId: c.GetInt64("departmentId"),
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
