#!/bin/bash

# Generate self-signed SSL certificate for Nginx
# This is used for Cloudflare Full SSL mode

CERT_DIR="./ssl"
mkdir -p "$CERT_DIR"

echo "🔐 Generating self-signed SSL certificate for notspies.org..."

openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout "$CERT_DIR/key.pem" \
    -out "$CERT_DIR/cert.pem" \
    -subj "/C=US/ST=State/L=City/O=Organization/CN=notspies.org" \
    -addext "subjectAltName=DNS:notspies.org,DNS:www.notspies.org"

echo "✅ SSL certificate generated at $CERT_DIR/cert.pem"
echo "✅ SSL key generated at $CERT_DIR/key.pem"
