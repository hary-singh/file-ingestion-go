# test-files/scripts/test-setup.sh
#!/bin/bash
set -e

wait_for_healthy() {
    local service=$1
    local max_attempts=30
    local attempt=1

    echo "Waiting for $service to be healthy..."
    while [ $attempt -le $max_attempts ]; do
        if docker-compose ps | grep -q "$service.*healthy"; then
            echo "$service is healthy"
            return 0
        fi
        echo "Attempt $attempt/$max_attempts..."
        sleep 2
        attempt=$((attempt + 1))
    done
    echo "$service failed to become healthy"
    docker-compose logs $service
    return 1
}

# Azure Storage connection string for local testing
export STORAGE_CONN="DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://localhost:10000/devstoreaccount1"

setup_services() {
    echo "Ensuring services are running..."
    wait_for_healthy kafka || exit 1
    wait_for_healthy postgres || exit 1
    wait_for_healthy schema-registry || exit 1

    # Initialize schemas
    # Get the absolute path to the scripts directory
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
    chmod +x "${SCRIPT_DIR}/init-schemas.sh"
    "${SCRIPT_DIR}/init-schemas.sh"

    # Create test container
    echo "Creating test container..."
    az storage container create \
        --name "cust123-transactions" \
        --connection-string "$STORAGE_CONN" \
        > /dev/null
}