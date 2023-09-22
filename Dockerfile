FROM docker.io/library/golang:1.20 as builder

WORKDIR /app

# Copy source code
COPY go.mod .
COPY go.sum .
COPY vendor/ vendor/
COPY cmd/ cmd/
COPY pkg/ pkg/

RUN go build -o bin/kafka-clickhouse-example cmd/main.go

# final stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.2

COPY --from=builder /app/bin/kafka-clickhouse-example /app/

ENTRYPOINT ["/app/kafka-clickhouse-example"]
