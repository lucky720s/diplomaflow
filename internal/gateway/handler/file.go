package handler

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	filev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/file/v1"
)

// abortIfBodyTooLarge returns true (and writes a JSON 413) when err indicates
// the request body exceeded the gateway upload limit (http.MaxBytesReader).
func abortIfBodyTooLarge(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) || strings.Contains(err.Error(), "request body too large") {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large"})
		return true
	}
	return false
}

func (h *Handler) UploadFile(c *gin.Context) {
	f, header, err := c.Request.FormFile("file")
	if err != nil {
		if abortIfBodyTooLarge(c, err) {
			return
		}
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
	ext := strings.ToLower(filepath.Ext(header.Filename))
	mimeType := mime.TypeByExtension(ext)

	if mimeType == "" {
		switch ext {
		case ".docx":
			mimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		case ".doc":
			mimeType = "application/msword"
		case ".pdf":
			mimeType = "application/pdf"
		case ".xlsx":
			mimeType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		case ".xls":
			mimeType = "application/vnd.ms-excel"
		case ".pptx":
			mimeType = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
		case ".ppt":
			mimeType = "application/vnd.ms-powerpoint"
		case ".txt":
			mimeType = "text/plain; charset=utf-8"
		default:
			mimeType = "application/octet-stream"
		}
	}

	if err := stream.Send(&filev1.UploadFileRequest{ //nolint:govet
		Data: &filev1.UploadFileRequest_Info{
			Info: &filev1.FileInfo{
				FileType:  mimeType,
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

// DownloadFile serves a stored file with the correct MIME type and a
// Content-Disposition derived from query params:
//   - ?download=1            → attachment (forced download)
//   - ?disposition=inline    → inline for previewable types (PDF/images/text),
//     otherwise attachment fallback
//   - no params              → attachment (back-compat default)
func (h *Handler) DownloadFile(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	ctx := outgoingCtx(c)

	// Resolve name + MIME from file metadata AND enforce access control before
	// streaming. GetFileInfo runs the same CanAccessFile check the download RPC
	// does, but as a unary call its error (PermissionDenied/NotFound) surfaces
	// here — before any status/headers are written. The streaming DownloadFile
	// RPC reports access errors only on the first Recv(), by which point the
	// gateway has already committed a 200 + Content-Disposition, leaking an
	// empty "successful" attachment for forbidden/missing files. So we gate on
	// GetFileInfo and fail fast with the correct 403/404.
	info, infoErr := h.fileClient.GetFileInfo(ctx, &filev1.GetFileInfoRequest{Id: id})
	if infoErr != nil {
		MapGRPCError(c, infoErr)
		return
	}

	fileName := "file"
	mimeType := "application/octet-stream"
	if info != nil {
		if info.File != nil {
			if info.File.FileName != "" {
				fileName = info.File.FileName
			}
			if info.File.FileType != "" {
				mimeType = info.File.FileType
			}
		}
		if fileName == "file" && info.Name != "" {
			fileName = info.Name
		}
		if mimeType == "application/octet-stream" && info.FileType != "" {
			mimeType = info.FileType
		}
	}

	forceDownload := isTruthy(c.Query("download"))
	wantInline := strings.EqualFold(c.Query("disposition"), "inline")
	inline := wantInline && !forceDownload && isInlinePreviewable(mimeType)

	stream, err := h.fileClient.DownloadFile(ctx, &filev1.DownloadFileRequest{Id: id})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.Header("Content-Type", mimeType)
	c.Header("X-Content-Type-Options", "nosniff")
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	c.Header("Content-Disposition", disposition+"; "+contentDispositionFilename(fileName))
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

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes":
		return true
	}
	return false
}

// isInlinePreviewable reports whether a content type is safe to render inline
// in the browser. PDFs, images and plain text qualify; everything else
// (DOC/DOCX/etc.) falls back to attachment.
func isInlinePreviewable(mimeType string) bool {
	mt := strings.ToLower(strings.TrimSpace(mimeType))
	if i := strings.IndexByte(mt, ';'); i >= 0 {
		mt = strings.TrimSpace(mt[:i])
	}
	if mt == "application/pdf" || mt == "text/plain" {
		return true
	}
	return strings.HasPrefix(mt, "image/")
}

// contentDispositionFilename builds the filename parameters for a
// Content-Disposition header: a sanitized ASCII filename plus an RFC 5987
// filename* for non-ASCII names. CR/LF/quotes/control chars are stripped to
// prevent header injection.
func contentDispositionFilename(name string) string {
	ascii := sanitizeFilename(name)
	if ascii == "" {
		ascii = "file"
	}
	return fmt.Sprintf("filename=%q; filename*=UTF-8''%s", ascii, rfc5987Encode(name))
}

func sanitizeFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r == '\r' || r == '\n' || r == '"' || r == '\\':
			// drop header-breaking and quoting chars
		case r < 0x20 || r == 0x7f:
			// drop control chars
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// rfc5987Encode percent-encodes the UTF-8 bytes of s per RFC 5987, leaving the
// attr-char set unescaped.
func rfc5987Encode(s string) string {
	const attrChars = "!#$&+-.^_`|~"
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			strings.IndexByte(attrChars, c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
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
