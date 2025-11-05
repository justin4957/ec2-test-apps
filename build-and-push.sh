#!/bin/bash

# Build and push all Docker images to ECR
set -e

AWS_REGION=us-east-1
AWS_ACCOUNT_ID=310829530225
ECR_REGISTRY=${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

echo "🔐 Logging into ECR..."
aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${ECR_REGISTRY}

echo ""
echo "📦 Building and pushing slogan-server..."
cd slogan-server
docker build -t slogan-server:latest .
docker tag slogan-server:latest ${ECR_REGISTRY}/slogan-server:latest
docker push ${ECR_REGISTRY}/slogan-server:latest
cd ..

echo ""
echo "📦 Building and pushing error-generator..."
cd error-generator
docker build -t error-generator:latest .
docker tag error-generator:latest ${ECR_REGISTRY}/error-generator:latest
docker push ${ECR_REGISTRY}/error-generator:latest
cd ..

echo ""
echo "📦 Building and pushing location-tracker..."
cd location-tracker
docker build -t location-tracker:latest .
docker tag location-tracker:latest ${ECR_REGISTRY}/location-tracker:latest
docker push ${ECR_REGISTRY}/location-tracker:latest
cd ..

echo ""
echo "📦 Building and pushing code-fix-generator..."
cd code-fix-generator
docker build -t code-fix-generator:latest .
docker tag code-fix-generator:latest ${ECR_REGISTRY}/code-fix-generator:latest
docker push ${ECR_REGISTRY}/code-fix-generator:latest
cd ..

echo ""
echo "✅ All images built and pushed successfully!"
echo ""
echo "📋 Pushed images:"
echo "  - ${ECR_REGISTRY}/slogan-server:latest"
echo "  - ${ECR_REGISTRY}/error-generator:latest"
echo "  - ${ECR_REGISTRY}/location-tracker:latest"
echo "  - ${ECR_REGISTRY}/code-fix-generator:latest"
echo ""
echo "🚀 Ready to deploy with ./deploy-to-ec2.sh"
