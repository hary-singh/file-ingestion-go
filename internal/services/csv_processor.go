package services

import (
	"encoding/csv"
	"fmt"
	"go.uber.org/zap"
	"io"
	"validator-function/internal/domain"
	"validator-function/internal/logger"
)

// CSVProcessor handles parsing and processing of CSV files
type CSVProcessor struct {
	schemaRegistry SchemaRegistry
	schema         string
	validator      RecordValidator
	converter      RecordConverter
	log            *zap.Logger
}

// NewCSVProcessor creates a new CSV processor instance
func NewCSVProcessor(schemaRegistry SchemaRegistry, schema string, validator RecordValidator) *CSVProcessor {
	return &CSVProcessor{
		schemaRegistry: schemaRegistry,
		schema:         schema,
		validator:      validator,
		converter:      NewTransactionConverter(),
		log:            logger.With(zap.String("component", "csv_processor")),
	}
}

// ProcessContent reads and processes CSV content
func (p *CSVProcessor) ProcessContent(content io.Reader) ([][]byte, []interface{}, error) {
	logger.Info("starting CSV processing",
		zap.String("component", "csv_processor"))

	reader := csv.NewReader(content)
	reader.TrimLeadingSpace = true

	// Read headers
	headers, err := reader.Read()
	if err != nil {
		return nil, nil, &domain.BusinessError{
			Type:    domain.ErrInvalidFile,
			Message: "failed to read CSV headers",
			Err:     err,
		}
	}

	logger.Info("read CSV headers",
		zap.String("component", "csv_processor"),
		zap.Strings("headers", headers))

	if err := p.validator.ValidateHeaders(headers); err != nil {
		return nil, nil, err
	}

	records, messages, err := p.processRecords(reader, headers)
	if err != nil {
		p.log.Error("failed to process records", zap.Error(err))
		return nil, nil, err
	}

	return records, messages, nil
}

func (p *CSVProcessor) processRecords(reader *csv.Reader, headers []string) ([][]byte, []interface{}, error) {
	var records [][]byte
	var messages []interface{}
	rowNum := 1 // Start at 1 since headers were row 0

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, &domain.BusinessError{
				Type:    domain.ErrInvalidFile,
				Message: fmt.Sprintf("error reading row %d", rowNum),
				Err:     err,
			}
		}

		rowNum++
		record, err := p.processRow(headers, row, rowNum)
		if err != nil {
			p.log.Error("record validation failed",
				zap.Int("row", rowNum),
				zap.Error(err))
			messages = append(messages, fmt.Sprintf("Row %d: %s", rowNum, err.Error()))
			continue
		}

		// Convert record to Avro binary format
		binary, err := p.schemaRegistry.EncodeToBinary(p.schema, record)
		if err != nil {
			return nil, nil, &domain.BusinessError{
				Type:    domain.ErrInvalidSchema,
				Message: fmt.Sprintf("failed to encode row %d", rowNum),
				Err:     err,
			}
		}

		records = append(records, binary)
	}

	if len(records) == 0 {
		return nil, nil, &domain.BusinessError{
			Type:    domain.ErrInvalidFile,
			Message: "no valid records found in file",
		}
	}

	return records, messages, nil
}

func (p *CSVProcessor) processRow(headers []string, row []string, rowNum int) (map[string]interface{}, error) {
	if len(headers) != len(row) {
		return nil, &domain.BusinessError{
			Type:    domain.ErrInvalidFile,
			Message: fmt.Sprintf("column count mismatch in row %d: expected %d, got %d", rowNum, len(headers), len(row)),
		}
	}

	// Create record map from headers and values
	record := make(map[string]string)
	for i, header := range headers {
		record[header] = row[i]
	}

	// Validate record
	if err := p.validator.ValidateRecord(record, rowNum); err != nil {
		return nil, err
	}

	// Convert to native format
	native, err := p.converter.ConvertToNative(record)
	if err != nil {
		return nil, err
	}

	return native, nil
}
