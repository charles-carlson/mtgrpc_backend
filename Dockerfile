FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOARCH=arm64 go build -o server .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates && \
    adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /app/server .

USER appuser

EXPOSE 50051

CMD ["./server"]
