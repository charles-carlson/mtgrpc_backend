docker run --rm -d \
    -p 4566:4566 \
    -p 8000:8000 \
    --name localstack \
    localstack/localstack

aws dynamodb create-table \
      --table-name cards \
      --attribute-definitions \
        AttributeName=name,AttributeType=S \
        AttributeName=set_number,AttributeType=S \
        AttributeName=set,AttributeType=S \
      --global-secondary-indexes \
          "[{
            \"IndexName\": \"set-index\",
            \"KeySchema\": [{\"AttributeName\":\"set\",\"KeyType\":\"HASH\"}],
            \"Projection\":{\"ProjectionType\":\"ALL\"}
          }]" \
      --key-schema \
        AttributeName=name,KeyType=HASH \
        AttributeName=set_number,KeyType=RANGE \
      --billing-mode PAY_PER_REQUEST \
      --endpoint-url http://localhost:8000 \
      --region us-west-1

go run . -local
aws dynamodb list-tables --endpoint-url http://localhost:8000 --region us-west-1
