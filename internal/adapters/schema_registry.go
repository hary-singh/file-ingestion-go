package adapters

import (
	"fmt"
	"github.com/linkedin/goavro/v2"
	"sync"
)

type ConfluentSchemaRegistry struct {
	schemaCache map[string]*goavro.Codec
	mutex       sync.RWMutex // Add mutex for thread safety
}

func NewConfluentSchemaRegistry() *ConfluentSchemaRegistry {
	return &ConfluentSchemaRegistry{
		schemaCache: make(map[string]*goavro.Codec),
	}
}

func (r *ConfluentSchemaRegistry) GetSchema(subject string, version int) (string, error) {
	// TODO: In production, implement actual HTTP call to Confluent Schema Registry
	// This is a sample schema for testing
	sampleSchema := `{
	        "type": "record",
	        "name": "Order",
	        "fields": [
	            {"name": "order_id", "type": "string"},
	            {"name": "customer_name", "type": "string"},
	            {"name": "order_date", "type": "string"},
	            {"name": "total_amount", "type": "double"}
	        ]
	    }`
	return sampleSchema, nil
}

func (r *ConfluentSchemaRegistry) Validate(schema string, data []byte) error {
	codec, err := r.getCodec(schema)
	if err != nil {
		return fmt.Errorf("failed to get codec: %w", err)
	}

	// Decode the Avro binary data to validate it
	native, remaining, err := codec.NativeFromBinary(data)
	if err != nil {
		return fmt.Errorf("failed to decode Avro binary: %w", err)
	}

	// Check if there's any remaining data (indicates potential corruption)
	if len(remaining) > 0 {
		return fmt.Errorf("corrupt Avro data: %d bytes remaining", len(remaining))
	}

	// Optional: Verify the decoded data structure
	if _, ok := native.(map[string]interface{}); !ok {
		return fmt.Errorf("decoded data is not a valid record")
	}

	return nil
}

func (r *ConfluentSchemaRegistry) getCodec(schema string) (*goavro.Codec, error) {
	r.mutex.RLock()
	if codec, ok := r.schemaCache[schema]; ok {
		r.mutex.RUnlock()
		return codec, nil
	}
	r.mutex.RUnlock()

	// Create new codec
	codec, err := goavro.NewCodec(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create codec: %w", err)
	}

	// Cache the codec
	r.mutex.Lock()
	r.schemaCache[schema] = codec
	r.mutex.Unlock()

	return codec, nil
}

// Helper method to encode data to Avro binary format
func (r *ConfluentSchemaRegistry) EncodeToBinary(schema string, native interface{}) ([]byte, error) {
	codec, err := r.getCodec(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to get codec: %w", err)
	}

	binary, err := codec.BinaryFromNative(nil, native)
	if err != nil {
		return nil, fmt.Errorf("failed to encode to Avro binary: %w", err)
	}

	return binary, nil
}
