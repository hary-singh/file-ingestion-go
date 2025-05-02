package adapters

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"io"
	"validator-function/internal/domain"
)

type AzureBlobStorage struct {
	client *azblob.Client
}

func NewAzureBlobStorage(connectionString string) (*AzureBlobStorage, error) {
	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob client: %w", err)
	}

	return &AzureBlobStorage{
		client: client,
	}, nil
}

func (s *AzureBlobStorage) DownloadFile(ctx context.Context, customerID, fileName string) (io.ReadCloser, error) {
	containerName := fmt.Sprintf("%s-transactions", customerID)
	blobClient := s.client.ServiceClient().NewContainerClient(containerName).NewBlockBlobClient(fileName)

	response, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download blob: %w", err)
	}

	return response.Body, nil
}

func (s *AzureBlobStorage) GetFileMetadata(ctx context.Context, customerID, fileName string) (*domain.FileMetadata, error) {
	containerName := fmt.Sprintf("%s-orders", customerID)
	blobClient := s.client.ServiceClient().NewContainerClient(containerName).NewBlockBlobClient(fileName)

	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob properties: %w", err)
	}

	metadata := &domain.FileMetadata{
		CustomerID:      customerID,
		FileName:        fileName,
		FileSize:        *props.ContentLength,
		UploadTime:      *props.LastModified,
		ValidationState: *props.Metadata["validationState"],
		SchemaSubject:   *props.Metadata["schemaSubject"],
	}

	return metadata, nil
}

func (s *AzureBlobStorage) UploadFile(ctx context.Context, customerID, fileName string, content io.Reader) error {
	containerName := fmt.Sprintf("%s-orders", customerID)
	blobClient := s.client.ServiceClient().NewContainerClient(containerName).NewBlockBlobClient(fileName)

	_, err := blobClient.UploadStream(ctx, content, nil)
	if err != nil {
		return fmt.Errorf("failed to upload blob: %w", err)
	}

	return nil
}

func (s *AzureBlobStorage) Client() *azblob.Client {
	return s.client
}
