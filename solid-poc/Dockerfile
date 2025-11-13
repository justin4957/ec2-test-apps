# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY solid-poc/go.mod ./
RUN go mod download

# Copy source code
COPY solid-poc/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o solid-poc .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/solid-poc .

# Copy proof-of-concept directory
COPY proof-of-concept/ ./proof-of-concept/

# Expose port
EXPOSE 9090

# Run the application
CMD ["./solid-poc"]
