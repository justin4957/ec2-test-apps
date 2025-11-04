# AWS F1 FPGA Deployment Guide

Complete guide for deploying the rhythm-driven error generator on AWS F1 instances with FPGA acceleration!

![FPGA](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExZ2RrYnY3NWMyeWExMTc2aWF5cnRocnRhbW5renRvY2VsN2g2MzNjMSZlcD12MV9naWZzX3NlYXJjaCZjdD1n/l378BbHA2eQCQ5jDa/giphy.gif)

## ⚠️ Important Considerations

### Cost Analysis

AWS F1 instances are **expensive**:

| Instance Type | FPGA Cards | vCPUs | RAM | Price (us-east-1) |
|--------------|------------|-------|-----|-------------------|
| f1.2xlarge | 1 | 8 | 122 GB | ~$1.65/hour |
| f1.4xlarge | 2 | 16 | 244 GB | ~$3.30/hour |
| f1.16xlarge | 8 | 64 | 976 GB | ~$13.20/hour |

**For testing:** f1.2xlarge is sufficient (~$40/day if running 24/7)

**Cost-saving tips:**
- Use spot instances (50-90% discount)
- Stop instance when not testing
- Use t3.2xlarge for synthesis (~$0.33/hour) then deploy to F1

### Time Investment

- **Initial setup**: 2-4 hours
- **Model synthesis**: 30 minutes - 2 hours per model
- **Testing/debugging**: Variable
- **Total first deployment**: 4-8 hours

### Prerequisites

- AWS account with F1 instance access (may require limit increase)
- Familiarity with AWS CLI and EC2
- Understanding of FPGA concepts
- Budget for instance hours (~$10-50 for testing)

## Architecture

```
┌─────────────────────────────────────────────────────┐
│         AWS F1 Instance (f1.2xlarge)                │
│                                                     │
│  ┌─────────────────┐        ┌──────────────────┐  │
│  │  Python App     │        │  Xilinx FPGA     │  │
│  │  (Rhythm Svc)   │───────>│  Ultrascale+     │  │
│  │                 │  PCIe  │                  │  │
│  │  - Flask API    │        │  - Beat Detector │  │
│  │  - hls4ml       │        │  - 1-10μs latency│  │
│  │  - Load balancer│        │  - Parallel proc │  │
│  └─────────────────┘        └──────────────────┘  │
│           │                                         │
│           │ HTTP                                    │
└───────────┼─────────────────────────────────────────┘
            │
            ▼
   ┌─────────────────┐
   │ Error Generator │
   │  (Separate EC2) │
   └─────────────────┘
```

## Step-by-Step Deployment

### Phase 1: Development Environment Setup

#### 1.1 Launch Development Instance

Use a **t3.2xlarge** for synthesis (cheaper than F1):

```bash
# Create key pair if needed
aws ec2 create-key-pair \
  --key-name rhythm-fpga-key \
  --query 'KeyMaterial' \
  --output text > rhythm-fpga-key.pem
chmod 400 rhythm-fpga-key.pem

# Launch development instance (Amazon Linux 2)
aws ec2 run-instances \
  --image-id ami-0c55b159cbfafe1f0 \
  --instance-type t3.2xlarge \
  --key-name rhythm-fpga-key \
  --security-group-ids sg-YOUR_SG \
  --subnet-id subnet-YOUR_SUBNET \
  --block-device-mappings '[{"DeviceName":"/dev/xvda","Ebs":{"VolumeSize":100}}]' \
  --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=rhythm-fpga-dev}]'
```

#### 1.2 Install Xilinx Tools

SSH into the instance:
```bash
ssh -i rhythm-fpga-key.pem ec2-user@<INSTANCE_IP>
```

Install prerequisites:
```bash
# Update system
sudo yum update -y

# Install development tools
sudo yum groupinstall "Development Tools" -y
sudo yum install git python3 python3-pip -y

# Install AWS FPGA SDK
cd ~
git clone https://github.com/aws/aws-fpga.git
cd aws-fpga
source sdk_setup.sh
```

#### 1.3 Install hls4ml and Dependencies

```bash
# Create virtual environment
python3 -m venv ~/fpga-env
source ~/fpga-env/bin/activate

# Install Python dependencies
pip install --upgrade pip
pip install tensorflow==2.10.0
pip install hls4ml
pip install qkeras
pip install numpy scipy

# Install Vivado HLS (AWS provides this)
# Follow AWS FPGA Developer Guide for Vivado installation
```

### Phase 2: Model Development & Synthesis

#### 2.1 Create and Train Beat Detection Model

Create training script:
```python
# ~/train_beat_model.py
import tensorflow as tf
from tensorflow import keras
from qkeras import QDense, QActivation, quantized_bits
import numpy as np
import hls4ml

# Load or create training data
# For demo, we'll use synthetic data
def create_synthetic_data(num_samples=1000):
    """Create synthetic beat detection training data"""
    # 10 time steps, 3 features (onset, spectral centroid, rolloff)
    X = np.random.randn(num_samples, 10, 3).astype(np.float32)
    # Binary classification: beat or no beat
    y = np.random.randint(0, 2, (num_samples, 1)).astype(np.float32)
    return X, y

X_train, y_train = create_synthetic_data(1000)
X_test, y_test = create_synthetic_data(200)

# Create quantized model for FPGA
def create_fpga_model():
    """Create quantized model suitable for FPGA deployment"""

    # 8-bit quantization
    quantizer = quantized_bits(8, 0, alpha=1)

    model = keras.Sequential([
        # Simple feedforward network (RNN too complex for initial deployment)
        keras.layers.Flatten(input_shape=(10, 3)),

        QDense(16,
               kernel_quantizer=quantizer,
               bias_quantizer=quantizer,
               name='dense1'),
        QActivation('relu', name='relu1'),

        QDense(8,
               kernel_quantizer=quantizer,
               bias_quantizer=quantizer,
               name='dense2'),
        QActivation('relu', name='relu2'),

        QDense(1,
               kernel_quantizer=quantizer,
               bias_quantizer=quantizer,
               name='output'),
        QActivation('sigmoid', name='sigmoid')
    ])

    model.compile(
        optimizer='adam',
        loss='binary_crossentropy',
        metrics=['accuracy']
    )

    return model

# Train model
print("Creating and training model...")
model = create_fpga_model()
model.fit(X_train, y_train,
          epochs=10,
          batch_size=32,
          validation_data=(X_test, y_test))

# Save model
model.save('beat_detector_model.h5')
print("Model saved!")

# Test accuracy
loss, accuracy = model.evaluate(X_test, y_test)
print(f"Test accuracy: {accuracy:.4f}")
```

Run training:
```bash
source ~/fpga-env/bin/activate
python3 ~/train_beat_model.py
```

#### 2.2 Convert Model to HLS

Create conversion script:
```python
# ~/convert_to_hls.py
import hls4ml
from tensorflow import keras
import numpy as np

print("Loading trained model...")
model = keras.models.load_model('beat_detector_model.h5')

print("Configuring hls4ml...")
config = hls4ml.utils.config_from_keras_model(model, granularity='name')

# Configure for AWS F1 FPGA (Xilinx Ultrascale+)
config['Model']['Precision'] = 'ap_fixed<16,6>'
config['Model']['ReuseFactor'] = 1  # Parallel implementation
config['Model']['Strategy'] = 'Latency'  # Optimize for low latency
config['Model']['BramFactor'] = 100000
config['Model']['Compression'] = False

# Set layer-specific precision
for layer in config['LayerName'].keys():
    config['LayerName'][layer]['Precision'] = 'ap_fixed<16,6>'

print("Converting to HLS...")
hls_model = hls4ml.converters.convert_from_keras_model(
    model,
    hls_config=config,
    output_dir='beat_detector_hls',
    fpga_part='xcvu9p-flgb2104-2-i',  # AWS F1 FPGA part
    backend='Vivado'
)

print("Compiling model...")
hls_model.compile()

print("Building (synthesis)...")
print("⚠️  This will take 30 minutes to 2 hours...")
report = hls_model.build(csim=True, synth=True, cosim=False, export=True)

print("\n" + "="*60)
print("SYNTHESIS REPORT")
print("="*60)
print(f"Latency: {report.get('LatencyMin', 'N/A')} - {report.get('LatencyMax', 'N/A')} cycles")
print(f"Interval: {report.get('IntervalMin', 'N/A')} - {report.get('IntervalMax', 'N/A')} cycles")
print(f"Clock Period: {report.get('TargetClockPeriod', 'N/A')} ns")
print("\nResource Usage:")
print(f"  BRAM: {report.get('BRAM_18K', 'N/A')}")
print(f"  DSP: {report.get('DSP48E', 'N/A')}")
print(f"  FF: {report.get('FF', 'N/A')}")
print(f"  LUT: {report.get('LUT', 'N/A')}")
print("="*60)

# Save report
with open('synthesis_report.txt', 'w') as f:
    f.write(str(report))

print("\n✓ Synthesis complete!")
print("Output directory: beat_detector_hls")
```

Run synthesis:
```bash
source ~/fpga-env/bin/activate
python3 ~/convert_to_hls.py

# This will take 30 minutes to 2 hours!
# Go get coffee ☕
```

### Phase 3: Deploy to F1

#### 3.1 Create AFI (Amazon FPGA Image)

```bash
cd beat_detector_hls/myproject_prj/solution1/impl/export

# Create tarball
tar -czf beat_detector.tar.gz *

# Upload to S3
aws s3 mb s3://rhythm-fpga-bucket
aws s3 cp beat_detector.tar.gz s3://rhythm-fpga-bucket/

# Create AFI
aws ec2 create-fpga-image \
  --name rhythm-beat-detector \
  --description "Beat detection model for rhythm-driven errors" \
  --input-storage-location Bucket=rhythm-fpga-bucket,Key=beat_detector.tar.gz \
  --logs-storage-location Bucket=rhythm-fpga-bucket,Key=logs

# Save the AFI ID from output
# This process takes 30-60 minutes
```

Check AFI status:
```bash
aws ec2 describe-fpga-images --fpga-image-ids afi-XXXXXXXXX
```

Wait until state is `available`.

#### 3.2 Launch F1 Instance

```bash
# Get AWS FPGA Developer AMI
aws ec2 run-instances \
  --image-id ami-FPGA_DEV_AMI \
  --instance-type f1.2xlarge \
  --key-name rhythm-fpga-key \
  --security-group-ids sg-YOUR_SG \
  --subnet-id subnet-YOUR_SUBNET \
  --iam-instance-profile Name=your-fpga-role \
  --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=rhythm-fpga-f1}]'
```

#### 3.3 Load AFI on F1

SSH to F1 instance:
```bash
ssh -i rhythm-fpga-key.pem centos@<F1_INSTANCE_IP>
```

Load the AFI:
```bash
# Set up AWS FPGA environment
source /opt/aws/aws-fpga/sdk_setup.sh

# Clear any existing AFI
sudo fpga-clear-local-image -S 0

# Load your AFI
sudo fpga-load-local-image -S 0 -I afi-XXXXXXXXX

# Verify
fpga-describe-local-image -S 0
```

### Phase 4: Deploy Rhythm Service

#### 4.1 Install Application

```bash
# Install Python and dependencies
sudo yum install python3 python3-pip git -y

# Clone your repository
cd ~
git clone https://github.com/justin4957/ec2-test-apps.git
cd ec2-test-apps/rhythm-service

# Install dependencies
pip3 install -r requirements.txt

# Copy synthesized model files
scp -i rhythm-fpga-key.pem \
  beat_detector_hls.tar.gz \
  centos@<F1_IP>:~/ec2-test-apps/rhythm-service/
```

#### 4.2 Configure for FPGA

Update `.env`:
```bash
PORT=5001
ERROR_GENERATOR_URL=http://error-generator-ec2:9090
USE_FPGA=true
FPGA_DEVICE=/dev/xdma0
```

#### 4.3 Create FPGA Interface

Create `fpga_driver.py`:
```python
import ctypes
import numpy as np

class FPGABeatDetector:
    """Driver for FPGA-accelerated beat detection"""

    def __init__(self, device_path='/dev/xdma0'):
        self.device = device_path
        # Load FPGA driver library
        self.lib = ctypes.CDLL('/opt/aws/lib/libfpga.so')
        self.setup_fpga()

    def setup_fpga(self):
        """Initialize FPGA connection"""
        # Initialize FPGA communication
        pass

    def predict(self, features):
        """Run inference on FPGA

        Args:
            features: Input features (10, 3) shape

        Returns:
            predictions: Beat probability
        """
        # Send data to FPGA via DMA
        # Receive results
        # Return predictions
        pass
```

#### 4.4 Run Service

```bash
cd ~/ec2-test-apps/rhythm-service
USE_FPGA=true python3 rhythm_service.py
```

### Phase 5: Benchmark Performance

Create benchmark script:
```python
# benchmark_fpga.py
import time
import numpy as np
from fpga_driver import FPGABeatDetector

def benchmark():
    fpga = FPGABeatDetector()

    # Test data
    test_data = np.random.randn(1000, 10, 3).astype(np.float32)

    print("Benchmarking FPGA performance...")

    # Warmup
    for i in range(10):
        fpga.predict(test_data[0])

    # Actual benchmark
    latencies = []
    for i in range(1000):
        start = time.perf_counter()
        fpga.predict(test_data[i])
        end = time.perf_counter()
        latencies.append((end - start) * 1_000_000)  # microseconds

    print("\n" + "="*60)
    print("FPGA PERFORMANCE RESULTS")
    print("="*60)
    print(f"Mean latency: {np.mean(latencies):.2f} μs")
    print(f"Median latency: {np.median(latencies):.2f} μs")
    print(f"Min latency: {np.min(latencies):.2f} μs")
    print(f"Max latency: {np.max(latencies):.2f} μs")
    print(f"Std dev: {np.std(latencies):.2f} μs")
    print(f"\nThroughput: {1_000_000 / np.mean(latencies):.0f} inferences/sec")
    print("="*60)

if __name__ == '__main__':
    benchmark()
```

Expected results:
```
Mean latency: 5-10 μs
Throughput: 100,000-200,000 inferences/sec
```

## Cost Optimization

### Use Spot Instances

```bash
# Request spot instance (60-90% discount!)
aws ec2 request-spot-instances \
  --spot-price "0.50" \
  --instance-count 1 \
  --type "one-time" \
  --launch-specification '{
    "ImageId": "ami-FPGA_AMI",
    "InstanceType": "f1.2xlarge",
    "KeyName": "rhythm-fpga-key",
    "SecurityGroupIds": ["sg-YOUR_SG"]
  }'
```

### Auto-Shutdown Script

```bash
# Add to crontab - shut down at night
# crontab -e
0 22 * * * sudo shutdown -h now

# Or use idle detection
*/5 * * * * [ $(uptime | awk '{print $10}' | cut -d, -f1) < 0.1 ] && sudo shutdown -h now
```

## Troubleshooting

### AFI Creation Fails

Check logs in S3:
```bash
aws s3 ls s3://rhythm-fpga-bucket/logs/
aws s3 cp s3://rhythm-fpga-bucket/logs/latest.log -
```

### FPGA Not Detected

```bash
# Check PCI devices
lspci | grep Xilinx

# Check FPGA status
fpga-describe-local-image -S 0

# Reload driver
sudo rmmod xdma
sudo modprobe xdma
```

### Performance Not as Expected

- Check resource utilization in synthesis report
- Increase parallelism (reduce ReuseFactor)
- Simplify model architecture
- Verify clock frequency

## Alternative: Simpler Testing Approach

If full F1 deployment is too complex, try:

### Option 1: Use C-Simulation

Test without FPGA hardware:
```bash
# In hls_model directory
hls_model.build(csim=True, synth=False)
# Simulates FPGA behavior on CPU
```

### Option 2: Use AWS F1 Developer Instance

AWS provides development instances with FPGA simulation:
- No actual FPGA hardware
- Free simulation environment
- Good for testing workflow

### Option 3: Start with CPU Profiling

Profile current CPU performance first:
```python
import cProfile
cProfile.run('beat_detector.detect_beats(audio_data)')
```

Then estimate FPGA gains based on bottlenecks.

## Expected Performance Gains

| Metric | CPU (TensorFlow) | FPGA (hls4ml) | Speedup |
|--------|------------------|---------------|---------|
| Latency | 10-50 ms | 1-10 μs | **1000-5000x** |
| Throughput | 20-100/sec | 100K-200K/sec | **1000-2000x** |
| Power | 65W TDP | 25W | **2.6x more efficient** |

## Summary Checklist

- [ ] Provision development instance (t3.2xlarge)
- [ ] Install Xilinx/AWS FPGA tools
- [ ] Train and quantize model
- [ ] Synthesize with hls4ml (30-120 min)
- [ ] Create AFI (30-60 min wait)
- [ ] Launch F1 instance
- [ ] Load AFI onto FPGA
- [ ] Deploy rhythm service
- [ ] Run benchmarks
- [ ] Compare CPU vs FPGA performance
- [ ] **Remember to stop instances!**

## Cost Estimate for Full Test

| Item | Time | Cost |
|------|------|------|
| Development instance | 4 hours | $1.32 |
| AFI creation | 1 hour | $0.00 |
| F1 testing | 2 hours | $3.30 |
| **Total** | **7 hours** | **~$4.62** |

With spot instances: **~$2-3 total**

## Next Steps

1. **Decision point**: Full FPGA deployment or simulation?
2. **If full deployment**: Follow Phase 1-5 above
3. **If simulation**: Use Option 1 (C-Simulation)
4. **Budget approval**: Get approval for F1 instance costs
5. **Time allocation**: Block 4-8 hours for initial setup

Would you like me to help you get started with any specific phase? 🚀
