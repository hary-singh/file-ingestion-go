# test-files/test-large.sh
#!/bin/bash
set -e

source "$(dirname "$0")/scripts/test-setup.sh"

setup_services

echo "Running large file test..."
az storage blob upload \
    --container-name "cust123-transactions" \
    --name "transactions-2024-03-20.csv" \
    --file "test-files/samples/large-transactions.csv" \
    --connection-string "$STORAGE_CONN" \
    > /dev/null

echo "Waiting for processing..."
sleep 10

echo "Running consumer to check messages..."
go run ./cmd/consumer/main.go

echo "Checking validator logs..."
docker-compose logs --tail=50 validator | grep "File processed successfully"