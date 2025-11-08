# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
COPY gcp-service-account.json ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o error-generator .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/error-generator .
COPY --from=builder /app/gcp-service-account.json .

CMD ["./error-generator"]
