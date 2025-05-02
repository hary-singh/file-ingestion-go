package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/linkedin/goavro/v2"
)

func main() {
	// Initialize Kafka consumer
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "test-consumer",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}
	defer consumer.Close()

	// Subscribe to topic
	err = consumer.SubscribeTopics([]string{"transactions.raw"}, nil)
	if err != nil {
		fmt.Printf("Failed to subscribe: %s\n", err)
		os.Exit(1)
	}

	// Load Avro schema from file
	schemaBytes, err := ioutil.ReadFile("schemas/transactions-v1.avsc")
	if err != nil {
		fmt.Printf("Failed to read schema file: %v\n", err)
		os.Exit(1)
	}
	codec, err := goavro.NewCodec(string(schemaBytes))
	if err != nil {
		fmt.Printf("Failed to create codec: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Starting to consume messages...")
	for {
		msg, err := consumer.ReadMessage(time.Second)
		if err != nil {
			if !err.(kafka.Error).IsTimeout() {
				fmt.Printf("Error reading message: %v\n", err)
			}
			continue
		}

		fmt.Printf("\nMessage at offset %d:\n", msg.TopicPartition.Offset)
		fmt.Printf("Key: %s\n", string(msg.Key))

		fmt.Println("Headers:")
		for _, header := range msg.Headers {
			fmt.Printf("  %s: %s\n", header.Key, string(header.Value))
		}

		// Verify message length for Confluent wire format
		if len(msg.Value) < 6 {
			fmt.Printf("Message too short: %d bytes\n", len(msg.Value))
			continue
		}

		// Verify magic byte
		if msg.Value[0] != 0 {
			fmt.Printf("Invalid magic byte: %d\n", msg.Value[0])
			continue
		}

		// Extract schema ID
		schemaID := binary.BigEndian.Uint32(msg.Value[1:5])

		// Decode Avro binary data
		native, _, err := codec.NativeFromBinary(msg.Value[5:])
		if err != nil {
			fmt.Printf("Error decoding Avro message (schema ID %d): %v\n", schemaID, err)
			continue
		}

		record, ok := native.(map[string]interface{})
		if !ok {
			fmt.Println("Invalid record format")
			continue
		}

		// Print main transaction fields
		printField := func(name string) {
			if value, exists := record[name]; exists {
				fmt.Printf("%s: %v\n", name, value)
			}
		}

		printField("transaction_id")
		printField("type")
		printField("patient_id")
		printField("provider_id")
		printField("date")

		// Print diagnosis codes
		if diagCodes, ok := record["diagnosis_codes"].([]interface{}); ok {
			fmt.Printf("Diagnosis Codes (%d):\n", len(diagCodes))
			for _, code := range diagCodes {
				if codeStr, ok := code.(string); ok {
					fmt.Printf("  %s\n", codeStr)
				}
			}
		}

		// Print line items
		if items, ok := record["items"].([]interface{}); ok && len(items) > 0 {
			fmt.Println("Items:")
			for i, item := range items {
				if lineItem, ok := item.(map[string]interface{}); ok {
					fmt.Printf("  Item %d:\n", i+1)
					fmt.Printf("    Product ID: %v\n", lineItem["product_id"])
					fmt.Printf("    HCPCS Code: %v\n", lineItem["hcpcs_code"])

					// Handle modifier union type
					if modifier, ok := lineItem["modifier"].(map[string]interface{}); ok {
						if modStr, ok := modifier["string"].(string); ok {
							fmt.Printf("    Modifier: %s\n", modStr)
						}
					} else {
						fmt.Printf("    Modifier: null\n")
					}

					fmt.Printf("    Quantity: %v\n", lineItem["quantity"])
					fmt.Printf("    Unit Price: %.2f\n", lineItem["unit_price"])
					fmt.Printf("    Supplier ID: %v\n", lineItem["supplier_id"])
				}
			}
		}

		// Print total amount and status
		if amount, ok := record["total_amount"].(float64); ok {
			fmt.Printf("Total Amount: %.2f\n", amount)
		}
		printField("status")
	}
}
