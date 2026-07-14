#!/usr/bin/env bash
set -euo pipefail

ENDPOINT="http://localhost:8000"
REGION="us-west-1"
TABLE="cards"
CONTAINER="dynamodb-local"

# DynamoDB Local namespaces data by region + access key. Export a consistent
# region and (dummy) credentials so BOTH the aws CLI below and the Go app
# (`go run`, via LoadDefaultConfig) hit the same namespace and see the table.
export AWS_REGION="$REGION"
export AWS_DEFAULT_REGION="$REGION"
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-local}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-local}"

# Start DynamoDB Local if it isn't already running; otherwise reuse it.
# -sharedDb: one DB shared across all clients regardless of creds/region.
# -inMemory: ephemeral, so each fresh container starts clean.
if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
    docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
    # Clear anything else still bound to 8000 (e.g. a leftover LocalStack container).
    stale="$(docker ps -aq --filter publish=8000)"
    [ -n "$stale" ] && docker rm -f $stale >/dev/null 2>&1 || true
    docker run --rm -d \
        -p 8000:8000 \
        --name "$CONTAINER" \
        amazon/dynamodb-local \
        -jar DynamoDBLocal.jar -sharedDb -inMemory >/dev/null
fi

# Wait for DynamoDB to accept requests before provisioning.
echo "waiting for DynamoDB Local..."
for i in $(seq 1 30); do
    if aws dynamodb list-tables --endpoint-url "$ENDPOINT" --region "$REGION" >/dev/null 2>&1; then
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "DynamoDB Local did not become ready in time" >&2
        exit 1
    fi
    sleep 1
done

# Recreate the table so the schema (incl. set-index) is always current.
aws dynamodb delete-table --table-name "$TABLE" \
    --endpoint-url "$ENDPOINT" --region "$REGION" >/dev/null 2>&1 || true

aws dynamodb create-table \
    --table-name "$TABLE" \
    --attribute-definitions \
        AttributeName=name,AttributeType=S \
        AttributeName=set_number,AttributeType=S \
        AttributeName=set,AttributeType=S \
    --global-secondary-indexes \
        '[{
            "IndexName": "set-index",
            "KeySchema": [{"AttributeName":"set","KeyType":"HASH"}],
            "Projection": {"ProjectionType":"ALL"}
        }]' \
    --key-schema \
        AttributeName=name,KeyType=HASH \
        AttributeName=set_number,KeyType=RANGE \
    --billing-mode PAY_PER_REQUEST \
    --endpoint-url "$ENDPOINT" \
    --region "$REGION"

# Verify the GSI exists before starting the server.
aws dynamodb describe-table --table-name "$TABLE" \
    --endpoint-url "$ENDPOINT" --region "$REGION" \
    --query 'Table.{Keys:KeySchema,GSI:GlobalSecondaryIndexes[].{name:IndexName,keys:KeySchema}}'

# Optionally seed the fresh table, then start the server.
# Usage: bash scripts/gen_local.sh [path/to/seed.(json|txt|csv)]
SEED="${1:-}"
if [ -n "$SEED" ]; then
    if [ ! -f "$SEED" ]; then
        echo "seed file not found: $SEED" >&2
        exit 1
    fi
    echo "seeding from $SEED"
    go run . -local -ingest "$SEED"
else
    go run . -local
fi
