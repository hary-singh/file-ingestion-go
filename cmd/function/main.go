package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"

	"validator-function/internal/adapters"
	"validator-function/internal/logger"
	"validator-function/internal/middleware"
	"validator-function/internal/services"
)

type blobEvent struct {
	EventType string        `json:"eventType"`
	Data      blobEventData `json:"data"`
}

type blobEventData struct {
	URL           string `json:"url"`
	ContentType   string `json:"contentType"`
	ContentLength int    `json:"contentLength"`
}

func main() {
	log := logger.With(
		zap.String("service", "file-validator"),
		zap.String("env", os.Getenv("ENVIRONMENT")))

	// Initialize services
	validator, err := initializeServices(log)
	if err != nil {
		log.Fatal("Failed to initialize services", zap.Error(err))
	}

	// Start health check server
	startHealthCheck()

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start file processing
	if os.Getenv("ENVIRONMENT") == "local" {
		go startLocalProcessing(validator, log)
	} else {
		go startEventGridProcessing(validator, log, stop)
	}

	// Wait for shutdown signal
	<-stop
	handleGracefulShutdown(log)
}

func initializeServices(log *zap.Logger) (*services.FileValidator, error) {
	// Initialize Kafka
	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BOOTSTRAP_SERVERS"),
		"security.protocol": "PLAINTEXT",
		"client.id":         "validator-producer",
	}
	producer, err := adapters.NewConfluentKafkaProducer(kafkaConfig, 3, time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	// Initialize other dependencies
	blobStorage, err := adapters.NewAzureBlobStorage(os.Getenv("AzureWebJobsStorage"))
	if err != nil {
		return nil, fmt.Errorf("failed to create blob storage: %w", err)
	}

	configRepo, err := adapters.NewPostgresConfigRepository(os.Getenv("DB_CONNECTION_STRING"))
	if err != nil {
		return nil, fmt.Errorf("failed to create config repository: %w", err)
	}

	return services.NewFileValidator(
		blobStorage,
		configRepo,
		producer,
		adapters.NewConfluentSchemaRegistry(),
	), nil
}

func startHealthCheck() {
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	go http.ListenAndServe(":8080", nil)
}

func startLocalProcessing(validator *services.FileValidator, log *zap.Logger) {
	// Wait for test-local.sh to create container and upload file
	time.Sleep(5 * time.Second)

	event := blobEvent{
		EventType: "Microsoft.Storage.BlobCreated",
		Data: blobEventData{
			URL:           "http://azurite:10000/devstoreaccount1/cust123-orders/orders-2024-03-20.csv",
			ContentType:   "text/csv",
			ContentLength: 1024,
		},
	}

	eventBytes, _ := json.Marshal(event)
	handler := middleware.RecoverMiddleware(createEventHandler(validator, log), log)

	if err := handler(context.Background(), eventBytes); err != nil {
		log.Error("Failed to process test file", zap.Error(err))
	}
}

func startEventGridProcessing(validator *services.FileValidator, log *zap.Logger, stop chan os.Signal) {
	handler := middleware.RecoverMiddleware(createEventHandler(validator, log), log)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			processEventGridEvents(handler, log)
		}
	}
}

func processEventGridEvents(handler func(context.Context, []byte) error, log *zap.Logger) {
	// TODO: Implement actual Event Grid event polling
	// This is a placeholder for the production Event Grid integration
}

func createEventHandler(validator *services.FileValidator, log *zap.Logger) func(context.Context, []byte) error {
	return func(ctx context.Context, data []byte) error {
		var event blobEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("invalid event data: %w", err)
		}

		customerID, fileName, err := parseEventURL(event.Data.URL)
		if err != nil {
			return fmt.Errorf("failed to parse event URL: %w", err)
		}

		result, err := validator.ProcessFile(ctx, customerID, fileName)
		if err != nil {
			log.Error("File processing failed",
				zap.Error(err),
				zap.String("customerID", customerID),
				zap.String("fileName", fileName))
			return err
		}

		log.Info("File processed successfully",
			zap.String("customerID", customerID),
			zap.String("fileName", fileName),
			zap.Int("recordsValid", result.RecordsValid),
			zap.Any("messages", result.Messages))

		return nil
	}
}

func parseEventURL(urlStr string) (customerID, fileName string, err error) {
	customerID = "cust123" // TODO: Extract from URL path
	fileName = "orders-2024-03-20.csv"
	return
}

func handleGracefulShutdown(log *zap.Logger) {
	log.Info("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	<-ctx.Done()
	log.Info("Shutdown complete")
}
