#!/bin/bash
set -e

echo "Initializing schemas..."

echo "Waiting for Schema Registry to be ready..."
until curl --silent --fail http://localhost:8081 > /dev/null; do
    echo "Waiting for Schema Registry..."
    sleep 1
done

echo "Registering schemas..."

# Convert schema file to single-line JSON string with proper escaping
SCHEMA=$(cat schemas/transactions-v1.avsc | tr -d '\n' | sed 's/"/\\"/g')

# Register Transactions schema with proper JSON payload
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST -H "Content-Type: application/json" \
  -d "{\"schema\":\"${SCHEMA}\"}" \
  http://localhost:8081/subjects/transactions-v1/versions)

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" -ne 200 ]; then
    echo "Schema registration failed with code $HTTP_CODE"
    echo "Error: $BODY"
    exit 1
fi

echo "Schemas registered successfully!"