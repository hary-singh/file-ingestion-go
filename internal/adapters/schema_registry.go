package adapters

import (
	"encoding/json"
	"fmt"
	"github.com/linkedin/goavro/v2"
	"io"
	"net/http"
	"sync"
)

type ConfluentSchemaRegistry struct {
	baseURL     string
	schemaCache map[string]*goavro.Codec
	mutex       sync.RWMutex
}

func NewConfluentSchemaRegistry() *ConfluentSchemaRegistry {
	return &ConfluentSchemaRegistry{
		baseURL:     "http://schema-registry:8081",
		schemaCache: make(map[string]*goavro.Codec),
	}
}

type schemaResponse struct {
	Schema string `json:"schema"`
}

func (r *ConfluentSchemaRegistry) GetSchema(subject string, version int) (string, error) {
	url := fmt.Sprintf("%s/subjects/%s/versions/%d", r.baseURL, subject, version)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch schema: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("schema registry returned status %d: %s", resp.StatusCode, body)
	}

	var schemaResp schemaResponse
	if err := json.NewDecoder(resp.Body).Decode(&schemaResp); err != nil {
		return "", fmt.Errorf("failed to decode schema response: %w", err)
	}

	return schemaResp.Schema, nil
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
