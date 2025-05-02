package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"
	"validator-function/internal/domain"
	"validator-function/internal/ports"
)

// SchemaRegistry defines the interface for schema validation and retrieval operations.
type SchemaRegistry interface {
	GetSchema(subject string, version int) (string, error)
	Validate(schema string, data []byte) error
	EncodeToBinary(schema string, native interface{}) ([]byte, error)
}

// FileValidator handles the validation and processing of uploaded files.
// It coordinates between storage, configuration, messaging, and schema validation.
type FileValidator struct {
	storage        ports.BlobStorage
	config         ports.ConfigRepository
	producer       ports.KafkaProducer
	schemaRegistry SchemaRegistry
}

// NewFileValidator creates a new instance of FileValidator with the required dependencies.
func NewFileValidator(
	storage ports.BlobStorage,
	config ports.ConfigRepository,
	producer ports.KafkaProducer,
	schemaRegistry SchemaRegistry,
) *FileValidator {
	return &FileValidator{
		storage:        storage,
		config:         config,
		producer:       producer,
		schemaRegistry: schemaRegistry,
	}
}

// ProcessFile handles the complete workflow of processing a customer's file:
// 1. Retrieves customer configuration
// 2. Downloads and parses the file
// 3. Validates records against schema
// 4. Publishes valid records to Kafka
// Returns a ValidationResult containing processing statistics.
func (v *FileValidator) ProcessFile(ctx context.Context, customerID, fileName string) (*domain.ValidationResult, error) {
	start := time.Now()
	result := &domain.ValidationResult{
		CustomerID: customerID,
		FileName:   fileName,
		StartTime:  start,
	}

	// Get customer configuration
	configStart := time.Now()
	config, err := v.config.GetCustomerConfig(ctx, customerID)
	if err != nil {
		return result, &domain.BusinessError{
			Type:    domain.ErrConfiguration,
			Message: "failed to get customer config",
			Err:     err,
		}
	}
	fmt.Printf("Config lookup took: %v\n", time.Since(configStart))

	// Get schema for validation
	schemaStart := time.Now()
	schema, err := v.schemaRegistry.GetSchema(config.SchemaSubject, 1)
	if err != nil {
		return result, &domain.BusinessError{
			Type:    domain.ErrInvalidSchema,
			Message: "failed to get schema",
			Err:     err,
		}
	}
	fmt.Printf("Schema lookup took: %v\n", time.Since(schemaStart))

	// Download and process file
	processStart := time.Now()
	records, messages, err := v.processFileContent(ctx, customerID, fileName, schema)
	if err != nil {
		return result, err
	}
	fmt.Printf("File processing took: %v\n", time.Since(processStart))

	// Store messages in result
	result.Messages = messages

	// Publish records to Kafka
	publishStart := time.Now()
	if err := v.publishRecords(ctx, records, customerID, fileName, config.KafkaTopic); err != nil {
		return result, err
	}
	kafkaTime := time.Since(publishStart)
	fmt.Printf("Kafka publishing took: %v (%.2f ms/record)\n",
		kafkaTime,
		float64(kafkaTime.Milliseconds())/float64(len(records)))

	result.RecordsTotal = len(records)
	result.RecordsValid = len(records)
	result.Status = "SUCCESS"
	result.Duration = time.Since(start).Seconds()

	fmt.Printf("Total processing time: %.2f seconds\n", result.Duration)
	fmt.Printf("Average time per record: %.2f ms\n",
		(result.Duration*1000)/float64(result.RecordsTotal))

	return result, nil
}

// processFileContent downloads and parses the file content, returning validated records.
func (v *FileValidator) processFileContent(ctx context.Context, customerID, fileName, schema string) ([][]byte, []interface{}, error) {
	fileContent, err := v.storage.DownloadFile(ctx, customerID, fileName)
	if err != nil {
		return nil, nil, &domain.BusinessError{
			Type:    domain.ErrStorage,
			Message: "failed to download file",
			Err:     err,
		}
	}
	defer fileContent.Close()

	return v.parseAndValidateCSV(fileContent, schema)
}

// parseAndValidateCSV reads a CSV file and converts records to Avro binary format.
func (v *FileValidator) parseAndValidateCSV(reader io.Reader, schema string) ([][]byte, []interface{}, error) {
	csvReader := csv.NewReader(reader)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, nil, &domain.BusinessError{
			Type:    domain.ErrInvalidFile,
			Message: "failed to read CSV",
			Err:     err,
		}
	}

	if len(records) < 2 {
		return nil, nil, &domain.BusinessError{
			Type:    domain.ErrInvalidFile,
			Message: "CSV file must contain headers and at least one row",
		}
	}

	return v.convertRecordsToAvro(records, schema)
}

// convertRecordsToAvro converts CSV records to Avro binary format with type conversion.
func (v *FileValidator) convertRecordsToAvro(records [][]string, schema string) ([][]byte, []interface{}, error) {
	headers := records[0]
	var results [][]byte
	var messages []interface{}

	fmt.Printf("Processing %d records...\n", len(records)-1)
	for i, record := range records[1:] {
		recordStart := time.Now()

		native, err := v.createNativeRecord(headers, record, i+1)
		if err != nil {
			return nil, nil, err
		}

		binary, err := v.schemaRegistry.EncodeToBinary(schema, native)
		if err != nil {
			return nil, nil, &domain.BusinessError{
				Type:    domain.ErrInvalidSchema,
				Message: fmt.Sprintf("row %d: failed to encode to Avro", i+1),
				Err:     err,
			}
		}

		messages = append(messages, native)
		results = append(results, binary)

		fmt.Printf("Record %d processed in: %.2f ms\n",
			i+1,
			float64(time.Since(recordStart).Microseconds())/1000.0)
	}

	return results, messages, nil
}

// createNativeRecord converts a CSV record to a native Go map with proper type conversion.
func (v *FileValidator) createNativeRecord(headers []string, record []string, rowNum int) (map[string]interface{}, error) {
	native := make(map[string]interface{}, len(headers))

	for j, value := range record {
		if j >= len(headers) {
			continue
		}

		var err error
		native[headers[j]], err = v.convertField(headers[j], value, rowNum)
		if err != nil {
			return nil, err
		}
	}

	return native, nil
}

// convertField handles type conversion for specific fields.
func (v *FileValidator) convertField(header, value string, rowNum int) (interface{}, error) {
	switch header {
	case "total_amount":
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, &domain.BusinessError{
				Type:    domain.ErrInvalidFile,
				Message: fmt.Sprintf("row %d: invalid float value for %s", rowNum, header),
				Err:     err,
			}
		}
		return val, nil
	case "order_date":
		val, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, &domain.BusinessError{
				Type:    domain.ErrInvalidFile,
				Message: fmt.Sprintf("row %d: invalid date value for %s", rowNum, header),
				Err:     err,
			}
		}
		return val.Format(time.RFC3339), nil
	default:
		return value, nil
	}
}

// publishRecords publishes the validated records to Kafka.
func (v *FileValidator) publishRecords(ctx context.Context, records [][]byte, customerID, fileName, topic string) error {
	headers := map[string]string{
		"x-customer-id":    customerID,
		"x-schema-version": "1",
		"x-ingestion-ts":   time.Now().UTC().Format(time.RFC3339),
		"x-file-name":      fileName,
	}

	for i, record := range records {
		if err := v.producer.PublishMessage(ctx, topic, customerID, record, headers); err != nil {
			return &domain.BusinessError{
				Type:    domain.ErrKafkaPublish,
				Message: fmt.Sprintf("failed to publish record %d", i),
				Err:     err,
			}
		}
	}

	return nil
}

func (v *FileValidator) GetBlobStorage() ports.BlobStorage {
	return v.storage
}
