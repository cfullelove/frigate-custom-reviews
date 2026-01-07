# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build the binary
RUN go build -o frigate-custom-reviews cmd/frigate-custom-reviews/main.go

# Runtime Stage
FROM alpine:3.19

WORKDIR /app

# Install necessary runtime dependencies (if any, though probably none for this static bin)
# apk add --no-cache ca-certificates is often good praxis
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/frigate-custom-reviews /app/frigate-custom-reviews
COPY config.yaml /app/config.yaml

CMD ["./frigate-custom-reviews", "-config", "/app/config.yaml"]
