package services

import (
	"io"
)

// RecordValidator defines the interface for validating file records
type RecordValidator interface {
	// ValidateHeaders checks if all required columns are present
	ValidateHeaders(headers []string) error

	// ValidateRecord validates a single record's content
	ValidateRecord(record map[string]string, rowNum int) error
}

// RecordConverter defines the interface for converting records to Avro format
type RecordConverter interface {
	// ConvertToNative converts a record to its native Go representation
	ConvertToNative(record map[string]string) (map[string]interface{}, error)
}

// FileProcessor defines the interface for processing file content
type FileProcessor interface {
	// ProcessContent processes the content of a file
	ProcessContent(content io.Reader) ([][]byte, []interface{}, error)
}
