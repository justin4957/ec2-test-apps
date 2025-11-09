#!/bin/bash

# Helper script to connect to EC2 instance via AWS Systems Manager Session Manager
# This eliminates the need for SSH keys and exposed port 22

set -e

AWS_REGION=us-east-1
INSTANCE_ID=i-04bd2369c252bee39

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🔐 Connecting to EC2 instance via AWS Systems Manager Session Manager...${NC}"
echo -e "${YELLOW}   Instance ID: ${INSTANCE_ID}${NC}"
echo -e "${YELLOW}   Region: ${AWS_REGION}${NC}"
echo ""

# Check if Session Manager plugin is installed
if ! command -v session-manager-plugin &> /dev/null; then
    echo -e "${YELLOW}⚠️  Session Manager plugin not found${NC}"
    echo ""
    echo "Please install the Session Manager plugin:"
    echo ""
    echo "macOS:"
    echo "  curl 'https://s3.amazonaws.com/session-manager-downloads/plugin/latest/mac_arm64/sessionmanager-bundle.zip' -o 'sessionmanager-bundle.zip'"
    echo "  unzip sessionmanager-bundle.zip"
    echo "  sudo ./sessionmanager-bundle/install -i /usr/local/sessionmanagerplugin -b /usr/local/bin/session-manager-plugin"
    echo ""
    echo "Linux:"
    echo "  curl 'https://s3.amazonaws.com/session-manager-downloads/plugin/latest/linux_64bit/session-manager-plugin.rpm' -o 'session-manager-plugin.rpm'"
    echo "  sudo yum install -y session-manager-plugin.rpm"
    echo ""
    echo "For other platforms, see:"
    echo "https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html"
    exit 1
fi

# Check if instance is online in SSM
PING_STATUS=$(aws ssm describe-instance-information \
    --filters "Key=InstanceIds,Values=${INSTANCE_ID}" \
    --region ${AWS_REGION} \
    --query 'InstanceInformationList[0].PingStatus' \
    --output text 2>/dev/null || echo "Unknown")

if [ "$PING_STATUS" != "Online" ]; then
    echo -e "${YELLOW}⚠️  Instance is not online in Systems Manager (Status: ${PING_STATUS})${NC}"
    echo ""
    echo "Possible causes:"
    echo "  1. SSM Agent not installed on instance"
    echo "  2. Instance IAM role missing AmazonSSMManagedInstanceCore policy"
    echo "  3. Instance has no internet connectivity"
    echo ""
    echo "Troubleshooting steps:"
    echo "  aws ssm describe-instance-information --filters \"Key=InstanceIds,Values=${INSTANCE_ID}\""
    echo ""
    exit 1
fi

echo -e "${GREEN}✅ Instance is online in Systems Manager${NC}"
echo ""

# Start session
echo -e "${GREEN}🚀 Starting interactive session...${NC}"
echo ""

aws ssm start-session \
    --target ${INSTANCE_ID} \
    --region ${AWS_REGION}
