# FPGA Deployment Tools

Helper scripts and documentation for deploying the rhythm service on AWS F1 FPGA instances.

## Quick Start

### 1. Estimate Costs

Run the cost estimator first to understand what you'll spend:

```bash
./cost-estimate.sh
```

This will ask about:
- How many hours you'll test
- Whether to use spot instances (60-90% discount!)
- Development time needed

Expected output:
```
Cost Breakdown (with 70% spot discount):
Development (t3.2xlarge): 4 hours × $0.1166 = $0.47
Testing (f1.2xlarge):     2 hours × $0.5775 = $1.16
------------------------------------------------
TOTAL ESTIMATED COST:                    $1.63
```

### 2. Provision F1 Instance

Once you've confirmed costs, provision an instance:

```bash
# On-demand (full price)
./provision-f1.sh

# Spot instance (60-90% discount!)
USE_SPOT=true ./provision-f1.sh
```

This will:
- Create SSH key pair
- Create security group
- Launch F1 instance
- Save connection info to `f1-instance-info.txt`

### 3. Follow Deployment Guide

See [`AWS_F1_DEPLOYMENT.md`](../AWS_F1_DEPLOYMENT.md) for complete step-by-step instructions.

## Files

| File | Purpose |
|------|---------|
| `provision-f1.sh` | Automates F1 instance provisioning |
| `cost-estimate.sh` | Estimates AWS costs before deployment |
| `README.md` | This file |

## Cost Comparison

| Scenario | Hours | Cost (On-Demand) | Cost (Spot) |
|----------|-------|------------------|-------------|
| Quick test | 2 | $3.30 | $0.50-1.50 |
| Half day | 4 | $6.60 | $1.00-3.00 |
| Full day | 8 | $13.20 | $2.00-6.00 |
| Development | 20 | $33.00 | $5.00-15.00 |

**Spot instances save 60-90%!**

## Performance Expectations

| Metric | CPU | FPGA | Improvement |
|--------|-----|------|-------------|
| Latency | 10-50 ms | 1-10 μs | 1000-5000x faster |
| Throughput | 20-100/sec | 100K-200K/sec | 1000-2000x more |
| Power | 65W | 25W | 2.6x more efficient |

## Decision Matrix

### ✅ Go with F1 if:
- You want **1000x+ performance boost**
- Budget allows $2-10 for testing
- You have 4-8 hours for setup
- You want to learn FPGA deployment
- You need ultra-low latency (<10μs)

### ⚠️ Consider alternatives if:
- Budget is tight (use C-simulation)
- Time is limited (CPU mode works)
- Just testing functionality
- Don't need microsecond latency

## Alternatives to Full F1 Deployment

### Option 1: C-Simulation (Free)
Test FPGA design without hardware:
```bash
# In your code
hls_model.build(csim=True, synth=False)
```
- ✅ Free
- ✅ ~90% accurate
- ✅ Fast to test
- ❌ No actual speedup

### Option 2: Vivado Co-Simulation
More accurate than C-sim:
```bash
hls_model.build(csim=True, synth=True, cosim=True)
```
- ✅ More accurate
- ✅ Finds timing issues
- ❌ Slower (hours)
- ❌ Still no real speedup

### Option 3: Cloud FPGA Providers
Other options besides AWS:
- Azure FPGAs
- Google Cloud TPUs
- Nimbix FPGA cloud
- Local FPGA boards (Xilinx, Intel)

## Safety Checklist

Before starting:
- [ ] AWS billing alerts configured
- [ ] Budget approved
- [ ] Time allocated (4-8 hours)
- [ ] SSH key pair created
- [ ] AWS credentials configured
- [ ] Cost estimate reviewed

During testing:
- [ ] Monitor AWS costs in console
- [ ] Set instance alarm for max runtime
- [ ] Take notes for documentation
- [ ] Benchmark performance properly

After testing:
- [ ] **STOP OR TERMINATE INSTANCE**
- [ ] Save benchmark results
- [ ] Save synthesized models
- [ ] Document learnings

## Common Commands

```bash
# Estimate costs
./cost-estimate.sh

# Provision F1 (spot)
USE_SPOT=true ./provision-f1.sh

# SSH to instance
ssh -i rhythm-fpga-key.pem ec2-user@<IP>

# Check FPGA status
fpga-describe-local-image -S 0

# Stop instance
aws ec2 stop-instances --instance-ids i-XXXXX

# Terminate instance (delete forever!)
aws ec2 terminate-instances --instance-ids i-XXXXX

# Check costs
aws ce get-cost-and-usage \
  --time-period Start=2025-11-01,End=2025-11-05 \
  --granularity DAILY \
  --metrics BlendedCost
```

## Troubleshooting

### "InsufficientInstanceCapacity"
F1 instances have limited capacity. Try:
- Different region
- Spot instances
- Different time of day
- Request limit increase

### "You have requested more instances than allowed"
Request limit increase:
- AWS Console → Service Quotas
- Request increase for F1 instances
- Usually approved in 24 hours

### Synthesis takes forever
Normal! FPGA synthesis is slow:
- Simple model: 30 min - 1 hour
- Complex model: 1-2 hours
- Very complex: 2-4 hours

Go get coffee ☕

### AFI creation pending
AFI creation takes time:
- Usually: 30-60 minutes
- Check status: `aws ec2 describe-fpga-images --fpga-image-ids afi-XXX`
- Wait until state is `available`

## Resources

- [AWS F1 Developer Guide](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/fpga-getting-started.html)
- [hls4ml Documentation](https://fastmachinelearning.org/hls4ml/)
- [Xilinx Vivado HLS](https://www.xilinx.com/products/design-tools/vivado/integration/esl-design.html)
- [AWS FPGA GitHub](https://github.com/aws/aws-fpga)

## Support

Having issues? Check:
1. [`AWS_F1_DEPLOYMENT.md`](../AWS_F1_DEPLOYMENT.md) - Full guide
2. [`TROUBLESHOOTING.md`](../rhythm-service/TROUBLESHOOTING.md) - Common issues
3. AWS F1 forum
4. hls4ml GitHub issues

## Next Steps

1. Run `./cost-estimate.sh` to understand costs
2. If budget allows, run `./provision-f1.sh`
3. Follow [`AWS_F1_DEPLOYMENT.md`](../AWS_F1_DEPLOYMENT.md)
4. Benchmark and document results!
5. **Stop instance when done!**

Happy FPGA hacking! 🚀
