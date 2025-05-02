# File-Based Ingestion System

A high-performance, modular ingestion pipeline for healthcare order and claim files. Built in Go, this system enables
secure file uploads, schema validation, and reliable streaming to Kafka for downstream analytics and processing.

## Features

- Upload via Azure Blob Storage or SFTP
- Per-customer Avro schema validation
- Kafka publishing with rich metadata headers
- Configurable via PostgreSQL
- Structured logging and tracing (Datadog, OpenTelemetry)
- Audit trail for every file processed
- Local development with Docker Compose

## Architecture

```mermaid
flowchart TD
    Customer[Customer]
    Blob[Azure Blob Storage]
    EventGrid[Azure Event Grid]
    Validator[Go Validator Function]
    ConfigDB[Customer Config DB]
    SchemaRegistry[Confluent Schema Registry]
    Kafka[Kafka]
    Downstream[Downstream Services]
    Audit[Audit Logs / Metrics]
    Customer --> Blob
    Blob --> EventGrid
    EventGrid --> Validator
    Validator --> Blob
    Validator --> ConfigDB
    Validator --> SchemaRegistry
    Validator --> Kafka
    Validator --> Audit
    Kafka --> Downstream
```

- **Azure Blob Storage**: Each customer uploads files to their own container.
- **Event Grid**: Triggers the Go validator function on new file uploads.
- **Go Validator Function**: Downloads, validates, and streams records to Kafka.
- **Confluent Schema Registry**: Stores and serves Avro schemas.
- **Kafka**: Decouples ingestion from downstream consumers.
- **PostgreSQL**: Stores customer configs and audit logs.

## Quickstart

### Prerequisites

- Docker & Docker Compose
- Azure CLI (for local blob upload)
- Go 1.24+ (for local builds)

### Local Development

1. Clone the repo
2. Start all services:

    ```shell
    make build
    make start
    ```
3. Running tests

    ```shell
    make test-small
    ```
   or
   ```shell
   make test-large
   ```
   or
    ```shell
   make test-all
   ```

4. The consumer runs automatically but and logs the messages to the console after deserialization.
   To inspect the consumer navigate here:
    ```shell
   cd cmd/consumer/main.go
   ```

   This builds the validator, starts all containers, uploads a sample file, and prints logs.


## Project Structure

- `cmd/function/` - Main entrypoint for the validator service
- `cmd/consumer/` - Kafka Avro consumer for testing
- `internal/services/` - Core validation, conversion, and processing logic
- `internal/adapters/` - Integrations for storage, Kafka, schema registry, config
- `internal/domain/` - Domain models and error types
- `test-files/` - Test scripts, sample data, and schema registration
- `schemas/` - Avro schema definitions

## Configuration

Set environment variables (see `docker-compose.yml` for examples):

- `AzureWebJobsStorage`
- `DB_CONNECTION_STRING`
- `KAFKA_BOOTSTRAP_SERVERS`
- `ENVIRONMENT` (set to `local` for local dev)

- The system expects CSV files to match the schema and validation profile for each customer.
- All Avro messages use the Confluent wire format.
- For local development, Azurite is used to emulate Azure Blob Storage.

## Example: Processing a File

1. Customer uploads a CSV to their Azure Blob container.
2. Event Grid triggers the Go validator.
3. Validator:
    - Loads customer config and Avro schema
    - Validates and encodes each record
    - Publishes to Kafka with headers (`x-customer-id`, `x-schema-version`, etc.)
    - Logs audit info to Postgres
4. Downstream consumers read from Kafka topics.

## Extending

- Add new file types or schemas by updating the config DB and schema registry.
- Add new downstream consumers by subscribing to Kafka topics.


