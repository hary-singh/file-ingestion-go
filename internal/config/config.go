package config

import (
	"fmt"
	"os"
)

type Config struct {
	StorageConnectionString string
	DBConnectionString      string
	KafkaBootstrapServers   string
	KafkaSaslUsername       string
	KafkaSaslPassword       string
	Environment             string
}

func ValidateConfig() (*Config, error) {
	required := []struct {
		name  string
		value string
	}{
		{"AzureWebJobsStorage", os.Getenv("AzureWebJobsStorage")},
		{"DB_CONNECTION_STRING", os.Getenv("DB_CONNECTION_STRING")},
		{"KAFKA_BOOTSTRAP_SERVERS", os.Getenv("KAFKA_BOOTSTRAP_SERVERS")},
		// Add other required vars
	}

	for _, r := range required {
		if r.value == "" {
			return nil, fmt.Errorf("required environment variable %s is not set", r.name)
		}
	}
	return &Config{
		StorageConnectionString: os.Getenv("AzureWebJobsStorage"),
		DBConnectionString:      os.Getenv("DB_CONNECTION_STRING"),
		KafkaBootstrapServers:   os.Getenv("KAFKA_BOOTSTRAP_SERVERS"),
		KafkaSaslUsername:       os.Getenv("KAFKA_SASL_USERNAME"),
		KafkaSaslPassword:       os.Getenv("KAFKA_SASL_PASSWORD"),
		Environment:             os.Getenv("ENVIRONMENT"),
	}, nil
}
