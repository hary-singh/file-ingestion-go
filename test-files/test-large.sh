# test-files/test-large.sh
#!/bin/bash
set -e

source "$(dirname "$0")/scripts/test-setup.sh"

setup_services

echo "Running large file test..."
az storage blob upload \
    --container-name "cust123-transactions" \
    --name "transactions-large-2024-03-20.csv" \
    --file "test-files/samples/large-transactions.csv" \
    --connection-string "$STORAGE_CONN" \
    > /dev/null

echo "Waiting for processing..."
sleep 10

echo "Checking Kafka messages..."
docker-compose exec -T kafka kafka-console-consumer \
    --bootstrap-server kafka:29092 \
    --topic transactions.raw \
    --from-beginning \
    --max-messages 2 \
    --timeout-ms 10000 \
    --property print.key=true \
    --property key.separator=":" \
    --property print.timestamp=true \
    --property print.headers=true \
    --property print.offset=true \
    --property key.deserializer=org.apache.kafka.common.serialization.StringDeserializer \
    --property value.deserializer=io.confluent.kafka.serializers.KafkaAvroDeserializer \
    --property schema.registry.url=http://schema-registry:8081