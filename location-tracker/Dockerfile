FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy source
COPY main.go ./

# Initialize Go module and build
RUN go mod init location-tracker && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o location-tracker .

# Production image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/location-tracker .

# Expose port
EXPOSE 8080

# Run
CMD ["./location-tracker"]
