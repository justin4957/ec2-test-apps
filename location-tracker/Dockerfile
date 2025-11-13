FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go module files and source
COPY go.mod go.sum* ./
COPY *.go ./
COPY types/ ./types/
COPY services/ ./services/

# Tidy dependencies and download
RUN go mod tidy && go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o location-tracker .

# Production image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/location-tracker .

# Expose ports
EXPOSE 8080 8443

# Run
CMD ["./location-tracker"]
