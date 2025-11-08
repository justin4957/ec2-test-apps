#!/bin/bash

# Build and push all Docker images to ECR
set -e

AWS_REGION=us-east-1
AWS_ACCOUNT_ID=310829530225
ECR_REGISTRY=${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

echo "🔐 Logging into ECR..."
aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${ECR_REGISTRY}

echo ""
echo "🏗️  Setting up buildx for multi-arch builds..."
docker buildx create --name multiarch --use 2>/dev/null || docker buildx use multiarch

echo ""
echo "📦 Building and pushing slogan-server..."
cd slogan-server
docker buildx build --platform linux/amd64 -t ${ECR_REGISTRY}/slogan-server:latest --push .
cd ..

echo ""
echo "📦 Building and pushing error-generator..."
cd error-generator
docker buildx build --platform linux/amd64 -t ${ECR_REGISTRY}/error-generator:latest --push .
cd ..

echo ""
echo "📦 Building and pushing location-tracker..."
cd location-tracker
docker buildx build --platform linux/amd64 -t ${ECR_REGISTRY}/location-tracker:latest --push .
cd ..

echo ""
echo "📦 Building and pushing code-fix-generator..."
cd code-fix-generator
docker buildx build --platform linux/amd64 -t ${ECR_REGISTRY}/code-fix-generator:latest --push .
cd ..

echo ""
echo "📦 Building and pushing nginx reverse proxy..."
cd nginx
docker buildx build --platform linux/amd64 -t ${ECR_REGISTRY}/nginx-proxy:latest --push .
cd ..

echo ""
echo "✅ All images built and pushed successfully!"
echo ""
echo "📋 Pushed images:"
echo "  - ${ECR_REGISTRY}/slogan-server:latest"
echo "  - ${ECR_REGISTRY}/error-generator:latest"
echo "  - ${ECR_REGISTRY}/location-tracker:latest"
echo "  - ${ECR_REGISTRY}/code-fix-generator:latest"
echo "  - ${ECR_REGISTRY}/nginx-proxy:latest"
echo ""
echo "🚀 Ready to deploy with ./deploy-to-ec2.sh"
