package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
		httpStatus = http.StatusBadRequest // 400
	case codes.NotFound:
		httpStatus = http.StatusNotFound // 404
	case codes.AlreadyExists:
		httpStatus = http.StatusConflict // 409
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden // 403
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized // 401
	case codes.DeadlineExceeded:
		httpStatus = http.StatusGatewayTimeout // 504
	case codes.Unavailable:
		httpStatus = http.StatusServiceUnavailable // 503
	}

	c.JSON(httpStatus, gin.H{
		"error": st.Message(),
		"code":  st.Code().String(),
	})
}
