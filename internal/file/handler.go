package file

import (
	"context"
	"io"
	"os"

	"github.com/lucky720s/diplomaflow/pkg/logger"
	filev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/file/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
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
	// first message must be info
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "failed to receive first message: %v", err)
	}
	info := req.GetInfo()
	if info == nil {
		return status.Error(codes.InvalidArgument, "first message must contain file info")
	}

	fileType := info.FileType
	originalName := info.FileName
	userID := info.UserId
	projectID := info.ProjectId

	id, tempPath, finalPath, f, err := h.service.StartUpload(originalName)
	if err != nil {
		h.logger.Error("failed to start upload", zap.Error(err))
		return status.Errorf(codes.Internal, "failed to start upload")
	}
	defer f.Close()

	var size int64

	// read chunks
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = os.Remove(tempPath)
			return status.Errorf(codes.Unknown, "stream recv error: %v", err)
		}

		chunk := req.GetChunk()
		if len(chunk) == 0 {
			continue
		}

		n, werr := f.Write(chunk)
		if werr != nil {
			_ = os.Remove(tempPath)
			return status.Errorf(codes.Internal, "failed to write chunk: %v", werr)
		}
		size += int64(n)
	}

	if err := h.service.CommitUpload(stream.Context(), id, tempPath, finalPath, userID, projectID, originalName, fileType, size); err != nil {
		h.logger.Error("commit upload failed", zap.Error(err))
		return status.Errorf(codes.Internal, "commit upload failed: %v", err)
	}

	h.logger.Info("File uploaded successfully", zap.String("id", id), zap.Int64("size", size))
	return stream.SendAndClose(&filev1.UploadFileResponse{
		Id:   id,
		Size: uint32(size),
	})
}

func (h *Handler) GetFileInfo(ctx context.Context, req *filev1.GetFileInfoRequest) (*filev1.GetFileInfoResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	path, meta, err := h.service.ResolveFilePath(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "file not found")
	}
	_ = path // we just confirm existence

	resp := &filev1.GetFileInfoResponse{
		Id:          req.Id,
		DownloadUrl: h.service.GetFileURL(req.Id),
	}

	// if metadata exists (stable mode)
	if meta != nil {
		resp.Id = meta.ID
		resp.Name = meta.FileName
		resp.Size = meta.Size
		resp.FileType = meta.FileType
		resp.DownloadUrl = h.service.GetFileURL(meta.ID)
	} else {
		// legacy: unknown original name
		resp.Name = req.Id
	}

	return resp, nil
}

func (h *Handler) DownloadFile(req *filev1.DownloadFileRequest, stream filev1.FileService_DownloadFileServer) error {
	if req.Id == "" {
		return status.Error(codes.InvalidArgument, "id is required")
	}

	path, _, err := h.service.ResolveFilePath(stream.Context(), req.Id)
	if err != nil {
		return status.Errorf(codes.NotFound, "file not found")
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
