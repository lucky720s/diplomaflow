package handler

import (
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	filev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/file/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func (h *Handler) CreateProject(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{"error": "project creation is not available via gateway; use supervisor approve/claim flow"})
}

func (h *Handler) GetProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	res, err := h.projectClient.GetProject(outgoingCtx(c), &projectv1.GetProjectRequest{ProjectId: id})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetStudentProjects(c *gin.Context) {
	studentID, err := strconv.ParseInt(c.Param("student_id"), 10, 64)
	if err != nil || studentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	res, err := h.projectClient.GetStudentProjects(outgoingCtx(c), &projectv1.GetStudentProjectsRequest{
		StudentId: studentID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListProjects(c *gin.Context) {
	studentID := c.GetInt64("userId")
	role := c.GetString("role")

	var req projectv1.GetStudentProjectsRequest
	if role == "student" {
		req.StudentId = studentID
	}

	res, err := h.projectClient.GetStudentProjects(outgoingCtx(c), &req)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) PerformProjectAction(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || projectID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	var req struct {
		ActionName string                 `json:"action_name" binding:"required"`
		Payload    map[string]interface{} `json:"payload"`
	}
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	payloadStruct, _ := structpb.NewStruct(req.Payload)

	res, err := h.projectClient.PerformAction(outgoingCtx(c), &projectv1.PerformActionRequest{
		ProjectId:  projectID,
		ActionName: req.ActionName,
		Payload:    payloadStruct,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// internal/gateway/handler/project.go — исправить UploadProjectDocument

func (h *Handler) UploadProjectDocument(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	userID := c.GetInt64("userId") // ← было "user_id", нужно "userId"

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	// 1. Загружаем файл через file_service (streaming)
	stream, err := h.fileClient.UploadFile(c.Request.Context())
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	ext := filepath.Ext(header.Filename)
	err = stream.Send(&filev1.UploadFileRequest{
		Data: &filev1.UploadFileRequest_Info{
			Info: &filev1.FileInfo{
				FileType:  ext,
				FileName:  header.Filename,
				UserId:    userID,
				ProjectId: projectID,
			},
		},
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	buf := make([]byte, 64*1024)
	for {
		n, readErr := file.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&filev1.UploadFileRequest{
				Data: &filev1.UploadFileRequest_Chunk{Chunk: buf[:n]},
			}); sendErr != nil {
				MapGRPCError(c, sendErr)
				return
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
			return
		}
	}

	fileResp, err := stream.CloseAndRecv()
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	// 2. Сабмитим документ через admin_service
	comment := c.Query("comment")
	resp, err := h.adminClient.SubmitDocument(outgoingCtx(c), &adminv1.SubmitDocumentRequest{
		ProjectId:   projectID,
		SubmittedBy: userID,
		FileIds:     []string{fileResp.Id},
		Comment:     comment,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file_id":       fileResp.Id,
		"submission_id": resp.SubmissionId,
		"message":       "Document uploaded and submitted successfully",
	})
}

func (h *Handler) SubmitProjectDocument(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	userID := c.GetInt64("userId")

	var req struct {
		FileIDs []string `json:"file_ids" binding:"required"`
		Comment string   `json:"comment"`
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.adminClient.SubmitDocument(outgoingCtx(c), &adminv1.SubmitDocumentRequest{
		ProjectId:   projectID,
		SubmittedBy: userID,
		FileIds:     req.FileIDs,
		Comment:     req.Comment,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
