package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
)

func (h *Handler) ListUsers(c *gin.Context) {
	universityID := c.GetInt64("universityId")
	role := c.Query("role")

	page := int32(1)
	pageSize := int32(20)

	if p := c.Query("page"); p != "" {
		// parse page
	}

	res, err := h.authClient.ListUsers(c.Request.Context(), &authv1.ListUsersRequest{
		UniversityId: universityID,
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
