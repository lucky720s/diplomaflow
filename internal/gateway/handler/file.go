package handler

import (
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	filev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/file/v1"
)

func (h *Handler) UploadFile(c *gin.Context) {
	f, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file provided"})
		return
	}
	defer f.Close()

	userID := c.GetInt64("userId")

	var projectID int64
	if q := c.Query("project_id"); q != "" {
		projectID, _ = strconv.ParseInt(q, 10, 64)
	}

	ctx := outgoingCtx(c)

	stream, err := h.fileClient.UploadFile(ctx)
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	ext := filepath.Ext(header.Filename)

	if err := stream.Send(&filev1.UploadFileRequest{ //nolint:govet
		Data: &filev1.UploadFileRequest_Info{
			Info: &filev1.FileInfo{
				FileType:  ext,
				FileName:  header.Filename,
				UserId:    userID,
				ProjectId: projectID,
			},
		},
	}); err != nil {
		MapGRPCError(c, err)
		return
	}

	buf := make([]byte, 64*1024)
	for {
		n, readErr := f.Read(buf)
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

	res, err := stream.CloseAndRecv()
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) DownloadFile(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	ctx := outgoingCtx(c)

	downloadName := "file"
	info, infoErr := h.fileClient.GetFileInfo(ctx, &filev1.GetFileInfoRequest{Id: id})
	if infoErr == nil && info != nil && info.Name != "" {
		downloadName = info.Name
	}

	stream, err := h.fileClient.DownloadFile(ctx, &filev1.DownloadFileRequest{Id: id})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	downloadName = strings.ReplaceAll(downloadName, "\n", "")
	downloadName = strings.ReplaceAll(downloadName, "\r", "")
	downloadName = strings.ReplaceAll(downloadName, `"`, "'")

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=\""+downloadName+"\"")
	c.Status(http.StatusOK)

	c.Stream(func(w io.Writer) bool {
		msg, recvErr := stream.Recv()
		if recvErr == io.EOF {
			return false
		}
		if recvErr != nil {
			return false
		}
		if len(msg.Chunk) > 0 {
			_, _ = w.Write(msg.Chunk)
		}
		return true
	})
}

func (h *Handler) DeleteFile(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	_, err := h.fileClient.DeleteFile(outgoingCtx(c), &filev1.DeleteFileRequest{
		Id:     id,
		UserId: c.GetInt64("userId"),
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
