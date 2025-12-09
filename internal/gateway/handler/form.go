package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	formv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/form/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func (h *Handler) SubmitForm(c *gin.Context) {
	var req struct {
		ProjectID int64                  `json:"project_id"`
		StepID    int64                  `json:"step_id"`
		Data      map[string]interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetInt64("userId")

	dataStruct, err := structpb.NewStruct(req.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data structure"})
		return
	}

	res, err := h.formClient.SubmitForm(c.Request.Context(), &formv1.SubmitFormRequest{
		ProjectId: req.ProjectID,
		StepId:    req.StepID,
		UserId:    userID,
		Data:      dataStruct,
	})

	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}
