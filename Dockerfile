# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build the binary
RUN go build -o frigate-stitcher cmd/frigate-stitcher/main.go

# Runtime Stage
FROM alpine:3.19

WORKDIR /app

# Install necessary runtime dependencies (if any, though probably none for this static bin)
# apk add --no-cache ca-certificates is often good praxis
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/frigate-stitcher /app/frigate-stitcher
COPY config.yaml /app/config.yaml

CMD ["./frigate-stitcher", "-config", "/app/config.yaml"]
