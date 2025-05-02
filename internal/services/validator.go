package services

import (
	"context"
	"fmt"
	"time"
	"validator-function/internal/domain"
	"validator-function/internal/logger"
	"validator-function/internal/ports"

	"go.uber.org/zap"
)

// SchemaRegistry defines interface for schema operations
type SchemaRegistry interface {
	GetSchema(subject string, version int) (string, error)
	Validate(schema string, data []byte) error
	EncodeToBinary(schema string, native interface{}) ([]byte, error)
}

// FileValidator handles the validation and processing of uploaded files
type FileValidator struct {
	storage         ports.BlobStorage
	config          ports.ConfigRepository
	producer        ports.KafkaProducer
	schemaRegistry  SchemaRegistry
	recordValidator RecordValidator
	log             *zap.Logger
}

// NewFileValidator creates a new FileValidator instance
func NewFileValidator(
	storage ports.BlobStorage,
	config ports.ConfigRepository,
	producer ports.KafkaProducer,
	schemaRegistry SchemaRegistry,
) *FileValidator {
	return &FileValidator{
		storage:         storage,
		config:          config,
		producer:        producer,
		schemaRegistry:  schemaRegistry,
		recordValidator: NewTransactionValidator(),
		log:             logger.With(zap.String("component", "validator")),
	}
}

// ProcessFile validates and processes a customer's uploaded file
func (v *FileValidator) ProcessFile(ctx context.Context, customerID, fileName string) (*domain.ValidationResult, error) {
	startTime := time.Now()
	result := &domain.ValidationResult{
		CustomerID: customerID,
		FileName:   fileName,
		StartTime:  startTime,
	}

	if err := v.validateInputs(customerID, fileName); err != nil {
		return v.failedResult(result, err)
	}

	config, schema, err := v.loadConfiguration(ctx, customerID)
	if err != nil {
		return v.failedResult(result, err)
	}

	records, messages, err := v.processContent(ctx, customerID, fileName, schema)
	if err != nil {
		return v.failedResult(result, err)
	}

	if err := v.publishRecords(ctx, records, customerID, fileName, config.KafkaTopic); err != nil {
		return v.failedResult(result, err)
	}

	return v.successResult(result, records, messages), nil
}

func (v *FileValidator) validateInputs(customerID, fileName string) error {
	if customerID == "" {
		return &domain.BusinessError{
			Type:    domain.ErrConfiguration,
			Message: "customer ID cannot be empty",
		}
	}
	if fileName == "" {
		return &domain.BusinessError{
			Type:    domain.ErrConfiguration,
			Message: "file name cannot be empty",
		}
	}
	return nil
}

func (v *FileValidator) loadConfiguration(ctx context.Context, customerID string) (*domain.CustomerConfig, string, error) {
	config, err := v.config.GetCustomerConfig(ctx, customerID)
	if err != nil {
		return nil, "", &domain.BusinessError{
			Type:    domain.ErrConfiguration,
			Message: "failed to get customer config",
			Err:     err,
		}
	}

	schema, err := v.schemaRegistry.GetSchema(config.SchemaSubject, 1)
	if err != nil {
		return nil, "", &domain.BusinessError{
			Type:    domain.ErrInvalidSchema,
			Message: "failed to get schema",
			Err:     err,
		}
	}

	return config, schema, nil
}

func (v *FileValidator) processContent(ctx context.Context, customerID, fileName, schema string) ([][]byte, []interface{}, error) {
	content, err := v.storage.DownloadFile(ctx, customerID, fileName)
	if err != nil {
		return nil, nil, &domain.BusinessError{
			Type:    domain.ErrStorage,
			Message: "failed to download file",
			Err:     err,
		}
	}
	defer content.Close()

	processor := NewCSVProcessor(v.schemaRegistry, schema, v.recordValidator)
	return processor.ProcessContent(content)
}

func (v *FileValidator) publishRecords(ctx context.Context, records [][]byte, customerID, fileName, topic string) error {
	headers := v.createMessageHeaders(customerID, fileName)

	for i, record := range records {
		if err := v.producer.PublishMessage(ctx, topic, customerID, record, headers); err != nil {
			return &domain.BusinessError{
				Type:    domain.ErrKafkaPublish,
				Message: fmt.Sprintf("failed to publish record %d", i),
				Err:     err,
			}
		}
		v.log.Info("Published message to Kafka",
			zap.String("topic", topic),
			zap.String("customerID", customerID),
			zap.Int("recordNumber", i+1),
			zap.ByteString("message", record))
	}
	return nil
}

func (v *FileValidator) createMessageHeaders(customerID, fileName string) map[string]string {
	return map[string]string{
		"x-customer-id":    customerID,
		"x-schema-version": "1",
		"x-ingestion-ts":   time.Now().UTC().Format(time.RFC3339),
		"x-file-name":      fileName,
	}
}

func (v *FileValidator) failedResult(result *domain.ValidationResult, err error) (*domain.ValidationResult, error) {
	result.Status = "FAILED"
	result.ErrorMessage = err.Error()
	result.Duration = time.Since(result.StartTime).Seconds()
	return result, err
}

func (v *FileValidator) successResult(result *domain.ValidationResult, records [][]byte, messages []interface{}) *domain.ValidationResult {
	result.RecordsTotal = len(records)
	result.RecordsValid = len(records)
	result.Status = "SUCCESS"
	result.Duration = time.Since(result.StartTime).Seconds()
	result.Messages = messages
	return result
}
