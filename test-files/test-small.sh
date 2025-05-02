#!/bin/bash
set -e

source "$(dirname "$0")/scripts/test-setup.sh"

setup_services

echo "Running small file test..."
az storage blob upload \
    --container-name "cust123-transactions" \
    --name "transactions-2024-03-20.csv" \
    --file "test-files/samples/valid-transactions.csv" \
    --connection-string "$STORAGE_CONN" \
    > /dev/null

echo "Waiting for processing..."
sleep 5

echo "Checking Kafka messages..."
docker-compose exec -T kafka \
    kafka-console-consumer \
    --bootstrap-server kafka:29092 \
    --topic transactions.raw \
    --from-beginning \
    --max-messages 1 \
    --timeout-ms 10000 \
    --property print.key=true \
    --property key.separator=":" \
    --property print.timestamp=true \
    --property print.headers=true

echo "Checking validator logs..."
docker-compose logs --tail=50 validator | grep "File processed successfully"