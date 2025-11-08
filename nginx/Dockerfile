FROM nginx:alpine

# Copy Nginx configuration
COPY nginx.conf /etc/nginx/nginx.conf

# Create SSL directory
RUN mkdir -p /etc/nginx/ssl

# Copy SSL certificates (will be generated before build)
COPY ssl/cert.pem /etc/nginx/ssl/cert.pem
COPY ssl/key.pem /etc/nginx/ssl/key.pem

# Expose ports
EXPOSE 80 443

# Start Nginx
CMD ["nginx", "-g", "daemon off;"]
