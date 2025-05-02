package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"validator-function/internal/domain"
)

var (
	transactionIDRegex = regexp.MustCompile(`^TRX\d{4,8}$`)
	patientIDRegex     = regexp.MustCompile(`^PT\d{4,8}$`)
	providerIDRegex    = regexp.MustCompile(`^PRV\d{2,4}$`)
	supplierIDRegex    = regexp.MustCompile(`^SUP\d{2,4}$`)
	validTypes         = map[string]bool{"ORDER": true, "CLAIM": true}
	validStatuses      = map[string]bool{"SUBMITTED": true, "PENDING": true, "APPROVED": true, "REJECTED": true}
)

type TransactionValidator struct{}

func NewTransactionValidator() RecordValidator {
	return &TransactionValidator{}
}

func (v *TransactionValidator) ValidateHeaders(headers []string) error {
	required := map[string]bool{
		"transaction_id": false,
		"type":           false,
		"patient_id":     false,
		"provider_id":    false,
		"date":           false,
		"quantity":       false,
		"unit_price":     false,
		"total_amount":   false,
		"status":         false,
	}

	for _, header := range headers {
		if _, exists := required[header]; exists {
			required[header] = true
		}
	}

	for header, found := range required {
		if !found {
			return &domain.BusinessError{
				Type:    domain.ErrInvalidFile,
				Message: fmt.Sprintf("required header missing: %s", header),
			}
		}
	}

	return nil
}

func (v *TransactionValidator) ValidateRecord(record map[string]string, rowNum int) error {
	errors := []error{
		validateTransactionID(record["transaction_id"], rowNum),
		validateType(record["type"], rowNum),
		validatePatientID(record["patient_id"], rowNum),
		validateProviderID(record["provider_id"], rowNum),
		validateDate(record["date"], rowNum),
		validateQuantity(record["quantity"], rowNum),
		validatePrice(record["unit_price"], "unit_price", rowNum),
		validatePrice(record["total_amount"], "total_amount", rowNum),
		validateStatus(record["status"], rowNum),
		validateSupplierID(record["supplier_id"], rowNum),
	}

	// Collect all validation errors
	var messages []string
	for _, err := range errors {
		if err != nil {
			messages = append(messages, err.Error())
		}
	}

	if len(messages) > 0 {
		return &domain.BusinessError{
			Type:    domain.ErrInvalidFile,
			Message: fmt.Sprintf("validation errors in row %d: %s", rowNum, strings.Join(messages, "; ")),
		}
	}

	return nil
}

// Add supplier ID validation
func validateSupplierID(id string, rowNum int) error {
	if !supplierIDRegex.MatchString(strings.TrimSpace(id)) {
		return fmt.Errorf("invalid supplier ID format (must be SUP followed by 2-4 digits)")
	}
	return nil
}

func validateTransactionID(id string, rowNum int) error {
	if !transactionIDRegex.MatchString(strings.TrimSpace(id)) {
		return fmt.Errorf("invalid transaction ID format (must be TRX followed by 6 digits)")
	}
	return nil
}

func validateType(transType string, rowNum int) error {
	if !validTypes[strings.ToUpper(strings.TrimSpace(transType))] {
		return fmt.Errorf("invalid transaction type (must be ORDER or CLAIM)")
	}
	return nil
}

func validatePatientID(id string, rowNum int) error {
	if !patientIDRegex.MatchString(strings.TrimSpace(id)) {
		return fmt.Errorf("invalid patient ID format (must be PT followed by 5 digits)")
	}
	return nil
}

func validateProviderID(id string, rowNum int) error {
	if !providerIDRegex.MatchString(strings.TrimSpace(id)) {
		return fmt.Errorf("invalid provider ID format (must be PRV followed by 3 digits)")
	}
	return nil
}

func validateDate(dateStr string, rowNum int) error {
	dateStr = strings.TrimSpace(dateStr)
	if _, err := time.Parse(time.RFC3339, dateStr); err != nil {
		return fmt.Errorf("invalid date format (must be RFC3339)")
	}
	return nil
}

func validateQuantity(quantityStr string, rowNum int) error {
	quantity, err := strconv.Atoi(strings.TrimSpace(quantityStr))
	if err != nil {
		return fmt.Errorf("invalid quantity (must be a positive integer)")
	}
	if quantity <= 0 {
		return fmt.Errorf("quantity must be greater than 0")
	}
	return nil
}

func validatePrice(priceStr, field string, rowNum int) error {
	price, err := strconv.ParseFloat(strings.TrimSpace(priceStr), 64)
	if err != nil {
		return fmt.Errorf("invalid %s (must be a positive number)", field)
	}
	if price <= 0 {
		return fmt.Errorf("%s must be greater than 0", field)
	}
	return nil
}

func validateStatus(status string, rowNum int) error {
	if !validStatuses[strings.ToUpper(strings.TrimSpace(status))] {
		return fmt.Errorf("invalid status (must be one of: SUBMITTED, PENDING, APPROVED, REJECTED)")
	}
	return nil
}
