package domain

import "time"

type FileMetadata struct {
	CustomerID      string    `json:"customer_id"`
	FileName        string    `json:"file_name"`
	FileSize        int64     `json:"file_size"`
	UploadTime      time.Time `json:"upload_time"`
	SchemaSubject   string    `json:"schema_subject"`
	ValidationState string    `json:"validation_state"`
}

type CustomerConfig struct {
	CustomerID        string `json:"customer_id"`
	SchemaSubject     string `json:"schema_subject"`
	KafkaTopic        string `json:"kafka_topic"`
	FilePattern       string `json:"file_pattern"`
	ValidationProfile string `json:"validation_profile"`
}

type ValidationResult struct {
	CustomerID   string        `json:"customer_id"`
	FileName     string        `json:"file_name"`
	RecordsTotal int           `json:"records_total"`
	RecordsValid int           `json:"records_valid"`
	StartTime    time.Time     `json:"start_time"`
	Duration     float64       `json:"duration_seconds"`
	Status       string        `json:"status"`
	ErrorMessage string        `json:"error_message,omitempty"`
	Messages     []interface{} `json:"messages,omitempty"`
}
