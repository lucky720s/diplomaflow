package handler

import (
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	filev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/file/v1"
)

func (h *Handler) UploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file provided"})
		return
	}
	defer file.Close()

	stream, err := h.fileClient.UploadFile(c.Request.Context())
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	err = stream.Send(&filev1.UploadFileRequest{
		Data: &filev1.UploadFileRequest_Info{
			Info: &filev1.FileInfo{FileType: filepath.Ext(header.Filename)},
		},
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	buffer := make([]byte, 1024*64)
	for {
		var n int
		n, err = file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
			return
		}

		err = stream.Send(&filev1.UploadFileRequest{
			Data: &filev1.UploadFileRequest_Chunk{
				Chunk: buffer[:n],
			},
		})
		if err != nil {
			MapGRPCError(c, err)
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
	if !isValidFileID(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	filePath := filepath.Join("/app/uploads", filepath.Clean(id))
	if !strings.HasPrefix(filePath, "/app/uploads/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	c.File(filePath)
}

func isValidFileID(id string) bool {
	matched, _ := regexp.MatchString(`^[a-f0-9\-]+(\.[a-z]+)?$`, id)
	return matched && !strings.Contains(id, "..")
}
