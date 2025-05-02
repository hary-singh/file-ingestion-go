package domain

import "fmt"

type ErrorType string

const (
	ErrInvalidFile   ErrorType = "INVALID_FILE"
	ErrInvalidSchema ErrorType = "INVALID_SCHEMA"
	ErrConfiguration ErrorType = "CONFIGURATION"
	ErrKafkaPublish  ErrorType = "KAFKA_PUBLISH"
	ErrStorage       ErrorType = "STORAGE"
)

type BusinessError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *BusinessError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}
