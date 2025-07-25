# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Копируем wait-for скрипт
COPY wait-for .
# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/main ./cmd/server/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for SSL connections
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .
# Expose port
EXPOSE 8080

# Run the binary
CMD ["./main"]
