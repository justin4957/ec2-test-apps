#!/bin/bash
# Cost estimation tool for AWS F1 deployment

echo "💰 AWS F1 Cost Estimator"
echo "========================="
echo ""

# Pricing (us-east-1)
F1_2XL_PRICE=1.65
F1_4XL_PRICE=3.30
T3_2XL_PRICE=0.3328

echo "Instance Pricing (us-east-1):"
echo "  f1.2xlarge:  \$$F1_2XL_PRICE/hour"
echo "  f1.4xlarge:  \$$F1_4XL_PRICE/hour"
echo "  t3.2xlarge:  \$$T3_2XL_PRICE/hour (for development)"
echo ""

# Get testing duration
read -p "How many hours will you test on F1? (default: 2): " TEST_HOURS
TEST_HOURS=${TEST_HOURS:-2}

read -p "Development hours on t3.2xlarge? (default: 4): " DEV_HOURS
DEV_HOURS=${DEV_HOURS:-4}

read -p "Use spot instances? (60-90% discount) (yes/no, default: yes): " USE_SPOT
USE_SPOT=${USE_SPOT:-yes}

# Calculate costs
if [ "$USE_SPOT" == "yes" ]; then
    SPOT_DISCOUNT=0.7  # 70% discount
    F1_COST=$(echo "$F1_2XL_PRICE * $TEST_HOURS * $SPOT_DISCOUNT" | bc)
    T3_COST=$(echo "$T3_2XL_PRICE * $DEV_HOURS * $SPOT_DISCOUNT" | bc)
    DISCOUNT_NOTE="(with 70% spot discount)"
else
    F1_COST=$(echo "$F1_2XL_PRICE * $TEST_HOURS" | bc)
    T3_COST=$(echo "$T3_2XL_PRICE * $DEV_HOURS" | bc)
    DISCOUNT_NOTE="(on-demand pricing)"
fi

TOTAL=$(echo "$F1_COST + $T3_COST" | bc)

echo ""
echo "Cost Breakdown $DISCOUNT_NOTE:"
echo "================================================"
printf "Development (t3.2xlarge): %d hours × \$%.4f = \$%.2f\n" \
    $DEV_HOURS $(echo "$T3_2XL_PRICE * $([ '$USE_SPOT' == 'yes' ] && echo 0.7 || echo 1)" | bc) $T3_COST
printf "Testing (f1.2xlarge):     %d hours × \$%.4f = \$%.2f\n" \
    $TEST_HOURS $(echo "$F1_2XL_PRICE * $([ '$USE_SPOT' == 'yes' ] && echo 0.7 || echo 1)" | bc) $F1_COST
echo "------------------------------------------------"
printf "TOTAL ESTIMATED COST:                    \$%.2f\n" $TOTAL
echo "================================================"
echo ""

# Timeline
echo "Estimated Timeline:"
echo "  1. Setup & development:    $DEV_HOURS hours"
echo "  2. Synthesis (automated):  1-2 hours"
echo "  3. AFI creation (wait):    0.5-1 hour"
echo "  4. F1 testing:             $TEST_HOURS hours"
TOTAL_TIME=$(echo "$DEV_HOURS + 2 + 1 + $TEST_HOURS" | bc)
echo "  ────────────────────────────────────"
echo "  TOTAL TIME:                ~$TOTAL_TIME hours"
echo ""

# Comparison
echo "Performance Expectations:"
echo "  CPU (current):     10-50 ms latency"
echo "  FPGA (expected):   1-10 μs latency"
echo "  Speedup:           1000-5000x faster! 🚀"
echo ""

# Recommendations
echo "💡 Recommendations:"
if (( $(echo "$TOTAL > 10" | bc -l) )); then
    echo "  ⚠️  Cost > \$10: Consider starting with C-simulation first"
    echo "     hls4ml supports CPU simulation without FPGA hardware"
fi

if [ "$USE_SPOT" != "yes" ]; then
    echo "  💰 Use spot instances to save 60-90%!"
fi

echo "  ✓ Stop instances immediately after testing"
echo "  ✓ Set up billing alerts in AWS console"
echo "  ✓ Use t3.2xlarge for development, f1 only for final testing"
echo ""

# Decision helper
echo "Should you proceed with F1?"
echo "────────────────────────────────────────────"
if (( $(echo "$TOTAL < 5" | bc -l) )); then
    echo "✅ LOW COST - Go for it!"
elif (( $(echo "$TOTAL < 15" | bc -l) )); then
    echo "⚠️  MODERATE COST - Proceed if budget allows"
else
    echo "❌ HIGH COST - Consider alternatives first"
    echo ""
    echo "Alternatives:"
    echo "  1. C-simulation (free, ~90% accurate)"
    echo "  2. Reduce testing time"
    echo "  3. Use spot instances"
fi
echo "────────────────────────────────────────────"
