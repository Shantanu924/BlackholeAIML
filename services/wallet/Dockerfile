# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN go mod tidy
RUN go build -o wallet .

# Production stage
FROM alpine:latest

# Install ca-certificates and curl for HTTPS & health checks
RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/wallet .

# Expose port 9000
EXPOSE 9000

# Run the wallet service
# BLOCKCHAIN_PEER_ID is resolved dynamically at runtime via wallet-entrypoint.sh
# Do not hardcode a Peer ID here — it changes on every fresh deployment
CMD ["./wallet", "-web", "-port", "9000"]
