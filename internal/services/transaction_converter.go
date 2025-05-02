package services

import (
	"fmt"
	"strconv"
	"strings"
	"validator-function/internal/domain"
)

type TransactionConverter struct{}

func NewTransactionConverter() RecordConverter {
	return &TransactionConverter{}
}

func (c *TransactionConverter) ConvertToNative(record map[string]string) (map[string]interface{}, error) {
	// Create the base transaction record
	native := make(map[string]interface{})

	// Convert required fields
	native["transaction_id"] = strings.TrimSpace(record["transaction_id"])
	native["type"] = strings.ToUpper(strings.TrimSpace(record["type"]))
	native["patient_id"] = strings.TrimSpace(record["patient_id"])
	native["provider_id"] = strings.TrimSpace(record["provider_id"])
	native["date"] = strings.TrimSpace(record["date"])
	native["status"] = strings.ToUpper(strings.TrimSpace(record["status"]))

	// Convert diagnosis codes to array
	diagnosisCodes := make([]string, 0)
	if codesStr := strings.TrimSpace(record["diagnosis_codes"]); codesStr != "" {
		diagnosisCodes = strings.Split(codesStr, ",")
		for i := range diagnosisCodes {
			diagnosisCodes[i] = strings.TrimSpace(diagnosisCodes[i])
		}
	}
	native["diagnosis_codes"] = diagnosisCodes

	// Convert numeric fields
	quantity, err := strconv.Atoi(strings.TrimSpace(record["quantity"]))
	if err != nil {
		return nil, &domain.BusinessError{
			Type:    domain.ErrInvalidFile,
			Message: "invalid quantity value",
			Err:     err,
		}
	}

	unitPrice, err := strconv.ParseFloat(strings.TrimSpace(record["unit_price"]), 64)
	if err != nil {
		return nil, &domain.BusinessError{
			Type:    domain.ErrInvalidFile,
			Message: "invalid unit price value",
			Err:     err,
		}
	}

	totalAmount, err := strconv.ParseFloat(strings.TrimSpace(record["total_amount"]), 64)
	if err != nil {
		return nil, &domain.BusinessError{
			Type:    domain.ErrInvalidFile,
			Message: "invalid total amount value",
			Err:     err,
		}
	}

	// Handle modifier as a union type
	var modifierValue interface{}
	modifier := strings.TrimSpace(record["modifier"])
	if modifier == "" {
		modifierValue = nil
	} else {
		// For union types, we need to specify which type we're using
		modifierValue = map[string]interface{}{
			"string": modifier,
		}
	}

	// Create line item with proper modifier handling
	lineItem := map[string]interface{}{
		"product_id":  strings.TrimSpace(record["product_id"]),
		"hcpcs_code":  strings.TrimSpace(record["hcpcs_code"]),
		"modifier":    modifierValue,
		"quantity":    quantity,
		"unit_price":  unitPrice,
		"supplier_id": strings.TrimSpace(record["supplier_id"]),
	}

	// Add all fields to the native record
	native["items"] = []map[string]interface{}{lineItem}
	native["total_amount"] = totalAmount

	// Verify all required fields are present
	if err := c.verifyRequiredFields(native); err != nil {
		return nil, err
	}

	return native, nil
}

func (c *TransactionConverter) verifyRequiredFields(native map[string]interface{}) error {
	required := []string{
		"transaction_id", "type", "patient_id", "provider_id",
		"date", "items", "total_amount", "status",
	}

	for _, field := range required {
		if value, exists := native[field]; !exists || value == nil {
			return &domain.BusinessError{
				Type:    domain.ErrInvalidFile,
				Message: fmt.Sprintf("missing required field: %s", field),
			}
		}
	}

	// Verify line items
	if items, ok := native["items"].([]map[string]interface{}); !ok || len(items) == 0 {
		return &domain.BusinessError{
			Type:    domain.ErrInvalidFile,
			Message: "at least one line item is required",
		}
	}

	return nil
}
