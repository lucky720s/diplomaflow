package file

import (
	"context"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/lucky720s/diplomaflow/pkg/logger"
	filev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/file/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Handler struct {
	filev1.UnimplementedFileServiceServer
	service *Service
	logger  *logger.Logger
}

func NewHandler(service *Service, log *logger.Logger) *Handler {
	return &Handler{service: service, logger: log}
}

func (h *Handler) UploadFile(stream filev1.FileService_UploadFileServer) error {
	// caller must be authenticated via metadata from gateway
	uid := callerUserID(stream.Context())
	if uid <= 0 {
		return status.Error(codes.Unauthenticated, "missing x-user-id")
	}

	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "failed to receive first message: %v", err)
	}
	info := req.GetInfo()
	if info == nil {
		return status.Error(codes.InvalidArgument, "first message must contain file info")
	}

	// SECURITY: не доверяем info.UserId от клиента вообще
	// (можешь также возвращать PermissionDenied если info.UserId != 0 && info.UserId != uid)
	info.UserId = uid

	id, tempPath, finalPath, f, err := h.service.StartUpload(info.FileName)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to start upload")
	}

	committed := false
	defer func() {
		_ = f.Close()
		if !committed {
			_ = os.Remove(tempPath)
			h.logger.Info("Upload cancelled, temp file cleaned",
				zap.String("id", id), zap.String("temp", tempPath))
		}
	}()

	maxBytes := h.service.MaxUploadBytes()
	var size int64

	for {
		chunkReq, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			return status.Errorf(codes.Canceled, "upload cancelled: %v", recvErr)
		}

		chunk := chunkReq.GetChunk()
		if len(chunk) == 0 {
			continue
		}

		size += int64(len(chunk))
		if size > maxBytes {
			return status.Errorf(codes.InvalidArgument, "file too large (max %d bytes)", maxBytes)
		}

		if _, werr := f.Write(chunk); werr != nil {
			return status.Errorf(codes.Internal, "failed to write chunk: %v", werr)
		}
	}

	// MIME is resolved inside CommitUpload from the committed file (passing "").
	meta, err := h.service.CommitUpload(
		stream.Context(),
		id, tempPath, finalPath,
		info.UserId, info.ProjectId,
		info.FileName, info.FileType, "", size,
	)
	if err != nil {
		return status.Errorf(codes.Internal, "commit upload failed: %v", err)
	}

	committed = true

	return stream.SendAndClose(&filev1.UploadFileResponse{
		Id:   id,
		Size: uint32(size),
		File: h.service.BuildFileRef(meta),
	})
}

func (h *Handler) GetFileInfo(ctx context.Context, req *filev1.GetFileInfoRequest) (*filev1.GetFileInfoResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	internal := isInternalCaller(ctx)
	uid := callerUserID(ctx)
	if uid <= 0 && !internal {
		return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
	}

	_, meta, err := h.service.ResolveFilePath(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "file not found")
	}

	// Legacy stored filenames (meta == nil) are only for internal callers.
	if meta == nil {
		if !internal {
			return nil, status.Error(codes.PermissionDenied, "legacy file ids are not accessible")
		}
		// Internal caller, legacy file: return the minimal info we have.
		return &filev1.GetFileInfoResponse{
			Id:          req.Id,
			Name:        req.Id,
			DownloadUrl: h.service.GetFileURL(req.Id),
		}, nil
	}

	// Authorization via user↔domain relation (internal callers are trusted).
	if !internal {
		allowed, accessErr := h.service.CanAccessFile(ctx, uid, callerRole(ctx), req.Id)
		if accessErr != nil {
			return nil, status.Errorf(codes.Internal, "access check failed")
		}
		if !allowed {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}
	}

	return &filev1.GetFileInfoResponse{
		Id:          meta.ID,
		Name:        meta.FileName,
		Size:        meta.Size,
		FileType:    metaMIME(meta),
		DownloadUrl: h.service.GetFileURL(meta.ID),
		File:        h.service.BuildFileRef(meta),
	}, nil
}

func (h *Handler) DownloadFile(req *filev1.DownloadFileRequest, stream filev1.FileService_DownloadFileServer) error {
	if req.Id == "" {
		return status.Error(codes.InvalidArgument, "id is required")
	}

	ctx := stream.Context()
	internal := isInternalCaller(ctx)
	uid := callerUserID(ctx)
	if uid <= 0 && !internal {
		return status.Error(codes.Unauthenticated, "missing x-user-id")
	}

	path, meta, err := h.service.ResolveFilePath(ctx, req.Id)
	if err != nil {
		return status.Errorf(codes.NotFound, "file not found")
	}

	// Legacy stored filenames are only for internal callers.
	if meta == nil {
		if !internal {
			return status.Error(codes.PermissionDenied, "legacy file ids are not accessible")
		}
	} else if !internal {
		allowed, accessErr := h.service.CanAccessFile(ctx, uid, callerRole(ctx), req.Id)
		if accessErr != nil {
			return status.Errorf(codes.Internal, "access check failed")
		}
		if !allowed {
			return status.Error(codes.PermissionDenied, "forbidden")
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to open file")
	}
	defer f.Close()

	buf := make([]byte, 64*1024)
	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			if serr := stream.Send(&filev1.DownloadFileResponse{Chunk: buf[:n]}); serr != nil {
				return status.Errorf(codes.Unknown, "failed to send chunk: %v", serr)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return status.Errorf(codes.Internal, "failed to read file: %v", rerr)
		}
	}

	return nil
}

func (h *Handler) DeleteFile(ctx context.Context, req *filev1.DeleteFileRequest) (*filev1.DeleteFileResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	uid := callerUserID(ctx)
	if uid <= 0 {
		return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
	}

	// SECURITY: не доверяем req.UserId
	if err := h.service.DeleteFile(ctx, req.Id, uid); err != nil {
		h.logger.Error("DeleteFile failed", zap.Error(err))
		// лучше маппить permission denied отдельно, но минимум так:
		return nil, status.Errorf(codes.Internal, "failed to delete file: %v", err)
	}

	h.logger.Info("File deleted", zap.String("id", req.Id), zap.Int64("user_id", uid))
	return &filev1.DeleteFileResponse{Success: true}, nil
}

func (h *Handler) CleanupOrphanedTempFiles(maxAge time.Duration) int {
	return h.service.CleanupOrphanedTempFiles(maxAge)
}
func callerUserID(ctx context.Context) int64 {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0
	}
	vals := md.Get("x-user-id")
	if len(vals) == 0 {
		return 0
	}
	id, _ := strconv.ParseInt(vals[0], 10, 64)
	return id
}

func callerRole(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	if vals := md.Get("x-user-role"); len(vals) > 0 {
		return vals[0]
	}
	return ""
}
func isInternalCaller(ctx context.Context) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || md == nil {
		return false
	}
	v := md.Get("x-internal-service")
	return len(v) > 0 && v[0] != ""
}
