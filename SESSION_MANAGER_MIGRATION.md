# AWS Systems Manager Session Manager Migration

This project has migrated from SSH-based access to **AWS Systems Manager Session Manager** for improved security and simplified access management.

## Benefits

### Security Improvements
- ✅ **No SSH Port Exposure**: Port 22 is no longer exposed to the internet
- ✅ **No SSH Key Management**: No need to manage, rotate, or secure SSH private keys
- ✅ **IAM-Based Access Control**: Access controlled through AWS IAM policies instead of security group rules
- ✅ **Session Logging**: All sessions automatically logged via CloudTrail for audit compliance
- ✅ **Session Recording**: Optional session recording for compliance requirements

### Operational Benefits
- ✅ **No IP Whitelist Management**: No need to constantly update security group rules for new IPs
- ✅ **Simplified Access**: Connect using AWS credentials without managing separate SSH keys
- ✅ **Better Auditing**: Built-in audit trail for compliance requirements
- ✅ **Port Forwarding**: Support for local port forwarding without SSH

## Prerequisites

### 1. Install the Session Manager Plugin

The Session Manager plugin must be installed on your local machine to use `aws ssm start-session`.

#### macOS (ARM64)
```bash
curl "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/mac_arm64/sessionmanager-bundle.zip" -o "sessionmanager-bundle.zip"
unzip sessionmanager-bundle.zip
sudo ./sessionmanager-bundle/install -i /usr/local/sessionmanagerplugin -b /usr/local/bin/session-manager-plugin
```

#### macOS (x86_64)
```bash
curl "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/mac/sessionmanager-bundle.zip" -o "sessionmanager-bundle.zip"
unzip sessionmanager-bundle.zip
sudo ./sessionmanager-bundle/install -i /usr/local/sessionmanagerplugin -b /usr/local/bin/session-manager-plugin
```

#### Linux (Amazon Linux 2 / RHEL / CentOS)
```bash
curl "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/linux_64bit/session-manager-plugin.rpm" -o "session-manager-plugin.rpm"
sudo yum install -y session-manager-plugin.rpm
```

#### Ubuntu / Debian
```bash
curl "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/ubuntu_64bit/session-manager-plugin.deb" -o "session-manager-plugin.deb"
sudo dpkg -i session-manager-plugin.deb
```

#### Windows
Download and run the installer:
```
https://s3.amazonaws.com/session-manager-downloads/plugin/latest/windows/SessionManagerPluginSetup.exe
```

#### Verify Installation
```bash
session-manager-plugin --version
```

### 2. AWS CLI Configuration

Ensure your AWS CLI is configured with credentials that have permissions to use Systems Manager:

```bash
aws configure
```

Required IAM permissions:
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ssm:StartSession",
                "ssm:TerminateSession",
                "ssm:ResumeSession",
                "ssm:DescribeSessions",
                "ssm:GetConnectionStatus"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "ssm:DescribeInstanceInformation",
                "ec2:DescribeInstances"
            ],
            "Resource": "*"
        }
    ]
}
```

## Usage

### Interactive Shell Access

Use the provided helper script for easy interactive access:

```bash
./ssm-connect.sh
```

Or use the AWS CLI directly:

```bash
aws ssm start-session \
    --target i-04bd2369c252bee39 \
    --region us-east-1
```

### Running Commands (Non-Interactive)

Use SSM `send-command` to run commands without an interactive session:

```bash
# View container logs
aws ssm send-command \
    --instance-ids i-04bd2369c252bee39 \
    --document-name "AWS-RunShellScript" \
    --parameters 'commands=["docker logs --tail 50 error-generator"]' \
    --query 'Command.CommandId' \
    --output text
```

### Deployment

The `deploy-to-ec2.sh` script now uses Session Manager instead of SSH:

```bash
./deploy-to-ec2.sh
```

The script will:
1. Check if the instance is online in Systems Manager
2. Create a deployment script with all environment variables
3. Execute the deployment via `aws ssm send-command`
4. Wait for completion and display output
5. Report success or failure

### View Logs

#### Option 1: Interactive Session
```bash
./ssm-connect.sh
# Then run:
docker logs -f error-generator
docker logs -f location-tracker
docker logs -f nginx-proxy
```

#### Option 2: Send Command
```bash
# Get command ID
COMMAND_ID=$(aws ssm send-command \
    --instance-ids i-04bd2369c252bee39 \
    --document-name "AWS-RunShellScript" \
    --parameters 'commands=["docker logs --tail 100 error-generator"]' \
    --query 'Command.CommandId' \
    --output text)

# Wait and get output
aws ssm wait command-executed \
    --command-id $COMMAND_ID \
    --instance-id i-04bd2369c252bee39

aws ssm get-command-invocation \
    --command-id $COMMAND_ID \
    --instance-id i-04bd2369c252bee39 \
    --query 'StandardOutputContent' \
    --output text
```

## EC2 Instance Configuration

The EC2 instance has been configured with:

1. **SSM Agent Installed**: The Amazon SSM Agent is installed and running
   ```bash
   sudo systemctl status amazon-ssm-agent
   ```

2. **IAM Role**: The instance has an IAM role (`EC2-ECR-Access`) with the following policies attached:
   - `AmazonSSMManagedInstanceCore` (for Session Manager)
   - `AmazonEC2ContainerRegistryReadOnly` (for pulling Docker images)
   - `AmazonS3FullAccess` (for S3 operations)

3. **SSM Agent Configuration**: The agent automatically registers with Systems Manager and maintains connectivity

## Security Group Changes

### Before (SSH-based access)
```
Inbound Rules:
- Port 22: Multiple individual IP addresses (20+ rules)
- Port 80: 0.0.0.0/0
- Port 443: 0.0.0.0/0
- Port 8080-8082: 0.0.0.0/0
```

### After (Session Manager)
```
Inbound Rules:
- Port 80: 0.0.0.0/0
- Port 443: 0.0.0.0/0
- Port 8080-8082: 0.0.0.0/0
# Port 22 can be completely removed!
```

### Recommended: Remove SSH Access

Once you've confirmed Session Manager is working, you can remove port 22 from the security group:

```bash
# Get security group ID
SG_ID=$(aws ec2 describe-instances \
    --instance-ids i-04bd2369c252bee39 \
    --query 'Reservations[0].Instances[0].SecurityGroups[0].GroupId' \
    --output text)

# Remove SSH rule (adjust CIDR as needed)
aws ec2 revoke-security-group-ingress \
    --group-id $SG_ID \
    --protocol tcp \
    --port 22 \
    --cidr 0.0.0.0/0
```

## Troubleshooting

### Instance Not Showing as "Online"

If the instance doesn't appear in Systems Manager:

1. **Check SSM Agent Status**:
   ```bash
   # Via SSH (if still available)
   ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@<instance-ip>
   sudo systemctl status amazon-ssm-agent

   # Restart if needed
   sudo systemctl restart amazon-ssm-agent
   ```

2. **Verify IAM Role**:
   ```bash
   aws iam list-attached-role-policies --role-name EC2-ECR-Access
   # Should include AmazonSSMManagedInstanceCore
   ```

3. **Check Instance Profile**:
   ```bash
   aws ec2 describe-instances \
       --instance-ids i-04bd2369c252bee39 \
       --query 'Reservations[0].Instances[0].IamInstanceProfile'
   ```

4. **Verify Internet Connectivity**:
   The instance needs internet access to communicate with Systems Manager endpoints. Check:
   - VPC route table has route to Internet Gateway
   - Network ACLs allow outbound HTTPS (port 443)
   - Security group allows outbound traffic

### Session Manager Plugin Not Found

If you get an error about the plugin not being found:

```
SessionManagerPlugin is not found. Please refer to SessionManager Documentation here: http://docs.aws.amazon.com/console/systems-manager/session-manager-plugin-not-found
```

Install the Session Manager plugin as described in [Prerequisites](#prerequisites).

### Permission Denied

If you get permission errors:

1. Check your AWS credentials:
   ```bash
   aws sts get-caller-identity
   ```

2. Verify you have the required IAM permissions (see [Prerequisites](#2-aws-cli-configuration))

3. Check CloudTrail logs for detailed error messages

## Migration Checklist

- [x] Install SSM Agent on EC2 instance
- [x] Attach `AmazonSSMManagedInstanceCore` policy to instance IAM role
- [x] Verify instance appears as "Online" in Systems Manager
- [x] Update `deploy-to-ec2.sh` to use SSM send-command
- [x] Create `ssm-connect.sh` helper script
- [x] Test deployment via Session Manager
- [x] Update documentation
- [ ] Remove port 22 from security group (after confirming Session Manager works)
- [ ] Remove SSH key from local machine (optional)
- [ ] Update team documentation/runbooks

## Additional Resources

- [AWS Systems Manager Session Manager Documentation](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager.html)
- [Installing the Session Manager Plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html)
- [Session Manager Prerequisites](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-prerequisites.html)
- [Logging Session Activity](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-logging.html)
