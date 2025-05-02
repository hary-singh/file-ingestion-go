package ports

import (
	"context"
	"io"
	"validator-function/internal/domain"
)

type BlobStorage interface {
	DownloadFile(ctx context.Context, customerID, fileName string) (io.ReadCloser, error)
	UploadFile(ctx context.Context, customerID, fileName string, content io.Reader) error
	GetFileMetadata(ctx context.Context, customerID, fileName string) (*domain.FileMetadata, error)
}

type ConfigRepository interface {
	GetCustomerConfig(ctx context.Context, customerID string) (*domain.CustomerConfig, error)
}
