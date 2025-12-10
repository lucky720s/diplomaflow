package file

import (
	"context"
	"io"

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
	return &Handler{
		service: service,
		logger:  log,
	}
}

func (h *Handler) UploadFile(stream filev1.FileService_UploadFileServer) error {
	var fileType string
	var fileSize uint32

	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "failed to receive first chunk: %v", err)
	}

	if info := req.GetInfo(); info != nil {
		fileType = info.FileType
	}

	file, fileID, err := h.service.CreateFile(fileType)
	if err != nil {
		h.logger.Error("failed to create file", zap.Error(err))
		return status.Errorf(codes.Internal, "failed to create file")
	}
	defer file.Close()

	if chunk := req.GetChunk(); len(chunk) > 0 {
		n, err := file.Write(chunk)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to write chunk: %v", err)
		}
		fileSize += uint32(n)
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Unknown, "stream error: %v", err)
		}

		chunk := req.GetChunk()
		if len(chunk) > 0 {
			n, err := file.Write(chunk)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to write chunk: %v", err)
			}
			fileSize += uint32(n)
		}
	}

	h.logger.Info("File uploaded successfully", zap.String("id", fileID), zap.Uint32("size", fileSize))

	return stream.SendAndClose(&filev1.UploadFileResponse{
		Id:   fileID,
		Size: fileSize,
	})
}

func (h *Handler) GetFileInfo(ctx context.Context, req *filev1.GetFileInfoRequest) (*filev1.GetFileInfoResponse, error) {
	if !h.service.FileExists(req.Id) {
		return nil, status.Errorf(codes.NotFound, "file not found")
	}

	meta, err := h.service.GetMetadata(ctx, req.Id)
	if err != nil {
		return &filev1.GetFileInfoResponse{
			Id:          req.Id,
			Name:        req.Id,
			DownloadUrl: h.service.GetFileURL(req.Id),
		}, nil
	}

	return &filev1.GetFileInfoResponse{
		Id:          req.Id,
		Name:        meta.FileName,
		DownloadUrl: h.service.GetFileURL(req.Id),
	}, nil
}
