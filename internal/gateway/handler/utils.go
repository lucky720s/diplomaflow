package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func parseInt64(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func MapGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	httpStatus := http.StatusInternalServerError

	switch st.Code() {
	case codes.OK:
		httpStatus = http.StatusOK
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.AlreadyExists:
		httpStatus = http.StatusConflict
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
	case codes.FailedPrecondition:
		httpStatus = http.StatusBadRequest
	case codes.DeadlineExceeded:
		httpStatus = http.StatusGatewayTimeout
	case codes.Unavailable:
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"error": st.Message(),
		"code":  st.Code().String(),
	})
}
func taskCtx(c *gin.Context) context.Context {
	userID := c.GetInt64("userId")

	role := c.GetString("role")
	if role == "" {
		role = "user"
	}
	if role == "user" {
		role = "student"
	}

	return metadata.NewOutgoingContext(
		c.Request.Context(),
		metadata.Pairs(
			"x-user-id", strconv.FormatInt(userID, 10),
			"x-user-role", role,
			"x-university-id", c.GetHeader("x-university-id"),
			"x-department-id", c.GetHeader("x-department-id"),
		),
	)
}
