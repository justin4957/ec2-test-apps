#!/bin/bash
# Provision AWS F1 instance for FPGA testing

set -e

echo "🚀 AWS F1 Instance Provisioning Script"
echo "======================================="
echo ""

# Configuration
KEY_NAME="${KEY_NAME:-rhythm-fpga-key}"
INSTANCE_TYPE="${INSTANCE_TYPE:-f1.2xlarge}"
REGION="${AWS_REGION:-us-east-1}"
USE_SPOT="${USE_SPOT:-false}"

# Check AWS CLI
if ! command -v aws &> /dev/null; then
    echo "❌ AWS CLI not found. Please install it first."
    exit 1
fi

# Check credentials
if ! aws sts get-caller-identity &> /dev/null; then
    echo "❌ AWS credentials not configured"
    exit 1
fi

echo "✓ AWS CLI configured"
echo "Region: $REGION"
echo "Instance type: $INSTANCE_TYPE"
echo "Key name: $KEY_NAME"
echo ""

# Create key pair if it doesn't exist
if ! aws ec2 describe-key-pairs --key-names "$KEY_NAME" &> /dev/null; then
    echo "Creating SSH key pair: $KEY_NAME..."
    aws ec2 create-key-pair \
        --key-name "$KEY_NAME" \
        --query 'KeyMaterial' \
        --output text > "${KEY_NAME}.pem"
    chmod 400 "${KEY_NAME}.pem"
    echo "✓ Key pair created: ${KEY_NAME}.pem"
else
    echo "✓ Key pair exists: $KEY_NAME"
fi

# Get default VPC
VPC_ID=$(aws ec2 describe-vpcs --filters "Name=isDefault,Values=true" --query "Vpcs[0].VpcId" --output text)
SUBNET_ID=$(aws ec2 describe-subnets --filters "Name=vpc-id,Values=$VPC_ID" --query "Subnets[0].SubnetId" --output text)

echo "✓ Using VPC: $VPC_ID"
echo "✓ Using Subnet: $SUBNET_ID"

# Create security group if needed
SG_NAME="rhythm-fpga-sg"
SG_ID=$(aws ec2 describe-security-groups --filters "Name=group-name,Values=$SG_NAME" --query "SecurityGroups[0].GroupId" --output text 2>/dev/null || echo "")

if [ -z "$SG_ID" ] || [ "$SG_ID" == "None" ]; then
    echo "Creating security group..."
    SG_ID=$(aws ec2 create-security-group \
        --group-name "$SG_NAME" \
        --description "Security group for rhythm FPGA instances" \
        --vpc-id "$VPC_ID" \
        --query 'GroupId' \
        --output text)

    # Allow SSH
    aws ec2 authorize-security-group-ingress \
        --group-id "$SG_ID" \
        --protocol tcp \
        --port 22 \
        --cidr 0.0.0.0/0

    # Allow rhythm service port
    aws ec2 authorize-security-group-ingress \
        --group-id "$SG_ID" \
        --protocol tcp \
        --port 5001 \
        --cidr 0.0.0.0/0

    echo "✓ Security group created: $SG_ID"
else
    echo "✓ Security group exists: $SG_ID"
fi

# Get FPGA Developer AMI (AL2)
AMI_ID=$(aws ec2 describe-images \
    --owners amazon \
    --filters "Name=name,Values=FPGA Developer AMI*" \
    --query 'Images | sort_by(@, &CreationDate) | [-1].ImageId' \
    --output text)

if [ -z "$AMI_ID" ] || [ "$AMI_ID" == "None" ]; then
    echo "⚠️  FPGA Developer AMI not found, using Amazon Linux 2"
    AMI_ID="ami-0c55b159cbfafe1f0"
fi

echo "✓ Using AMI: $AMI_ID"
echo ""

# Cost warning
if [ "$INSTANCE_TYPE" == "f1.2xlarge" ]; then
    COST="\$1.65/hour (~\$40/day)"
elif [ "$INSTANCE_TYPE" == "f1.4xlarge" ]; then
    COST="\$3.30/hour (~\$80/day)"
else
    COST="Check AWS pricing"
fi

echo "⚠️  COST WARNING"
echo "Instance type: $INSTANCE_TYPE"
echo "Estimated cost: $COST"
echo ""
read -p "Continue? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "Cancelled."
    exit 0
fi

echo ""
echo "Launching instance..."

# Launch instance
if [ "$USE_SPOT" == "true" ]; then
    echo "Using SPOT pricing (60-90% discount)"
    # Spot instance request
    SPOT_REQUEST=$(aws ec2 request-spot-instances \
        --spot-price "0.50" \
        --instance-count 1 \
        --type "one-time" \
        --launch-specification "{
            \"ImageId\": \"$AMI_ID\",
            \"InstanceType\": \"$INSTANCE_TYPE\",
            \"KeyName\": \"$KEY_NAME\",
            \"SecurityGroupIds\": [\"$SG_ID\"],
            \"SubnetId\": \"$SUBNET_ID\"
        }" \
        --query 'SpotInstanceRequests[0].SpotInstanceRequestId' \
        --output text)

    echo "Spot request ID: $SPOT_REQUEST"
    echo "Waiting for fulfillment..."

    aws ec2 wait spot-instance-request-fulfilled --spot-instance-request-ids "$SPOT_REQUEST"

    INSTANCE_ID=$(aws ec2 describe-spot-instance-requests \
        --spot-instance-request-ids "$SPOT_REQUEST" \
        --query 'SpotInstanceRequests[0].InstanceId' \
        --output text)
else
    # On-demand instance
    INSTANCE_ID=$(aws ec2 run-instances \
        --image-id "$AMI_ID" \
        --instance-type "$INSTANCE_TYPE" \
        --key-name "$KEY_NAME" \
        --security-group-ids "$SG_ID" \
        --subnet-id "$SUBNET_ID" \
        --block-device-mappings '[{"DeviceName":"/dev/xvda","Ebs":{"VolumeSize":100}}]' \
        --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=rhythm-fpga-f1}]" \
        --query 'Instances[0].InstanceId' \
        --output text)
fi

echo "✓ Instance launched: $INSTANCE_ID"
echo "Waiting for instance to be running..."

aws ec2 wait instance-running --instance-ids "$INSTANCE_ID"

# Get public IP
PUBLIC_IP=$(aws ec2 describe-instances \
    --instance-ids "$INSTANCE_ID" \
    --query 'Reservations[0].Instances[0].PublicIpAddress' \
    --output text)

echo ""
echo "✅ F1 Instance Ready!"
echo "======================================"
echo "Instance ID: $INSTANCE_ID"
echo "Public IP: $PUBLIC_IP"
echo "SSH Command: ssh -i ${KEY_NAME}.pem ec2-user@$PUBLIC_IP"
echo ""
echo "Next steps:"
echo "1. SSH to instance"
echo "2. Run: source /opt/aws/aws-fpga/sdk_setup.sh"
echo "3. Follow AWS_F1_DEPLOYMENT.md guide"
echo ""
echo "⚠️  Remember to stop instance when done!"
echo "Stop command: aws ec2 stop-instances --instance-ids $INSTANCE_ID"
echo "======================================"

# Save instance info
cat > f1-instance-info.txt <<EOF
Instance ID: $INSTANCE_ID
Public IP: $PUBLIC_IP
Instance Type: $INSTANCE_TYPE
Key: ${KEY_NAME}.pem
SSH: ssh -i ${KEY_NAME}.pem ec2-user@$PUBLIC_IP
Stop: aws ec2 stop-instances --instance-ids $INSTANCE_ID
Terminate: aws ec2 terminate-instances --instance-ids $INSTANCE_ID
EOF

echo "✓ Instance info saved to: f1-instance-info.txt"
