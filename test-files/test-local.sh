#!/bin/bash
set -e

# Helper function to wait for service health
wait_for_healthy() {
    local service=$1
    local max_attempts=30
    local attempt=1

    echo "Waiting for $service to be healthy..."
    while [ $attempt -le $max_attempts ]; do
        if docker-compose ps $service | grep -q "healthy"; then
            echo "$service is healthy"
            return 0
        fi
        docker-compose logs --tail 10 $service
        echo "Attempt $attempt/$max_attempts: $service not yet healthy, waiting..."
        sleep 2
        attempt=$((attempt + 1))
    done
    echo "$service failed to become healthy"
    docker-compose logs $service
    return 1
}

echo "Ensuring services are running..."

wait_for_healthy kafka || exit 1
wait_for_healthy postgres || exit 1
wait_for_healthy schema-registry || exit 1

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

chmod +x "${SCRIPT_DIR}/scripts/init-schemas.sh"
"${SCRIPT_DIR}/scripts/init-schemas.sh"

# Azure Storage connection string
STORAGE_CONN="DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://localhost:10000/devstoreaccount1"

echo "Creating test container..."
az storage container create \
  --name "cust123-orders" \
  --connection-string "$STORAGE_CONN" \
  > /dev/null

echo "Uploading test file..."
az storage blob upload \
  --container-name "cust123-orders" \
  --name "orders-2024-03-20.csv" \
  --file "test-files/samples/valid-orders.csv" \
  --connection-string "$STORAGE_CONN" \
  > /dev/null

echo "Waiting for processing..."
sleep 10

echo "Checking Kafka messages..."
docker-compose exec -T kafka kafka-console-consumer \
  --bootstrap-server kafka:29092 \
  --topic orders.raw \
  --from-beginning \
  --max-messages 2 \
  --timeout-ms 10000 \
  --property print.key=true \
  --property print.timestamp=true \
  2>/dev/null || echo "No messages found in topic"

echo "Checking validator logs..."
docker-compose logs --tail 50 validator