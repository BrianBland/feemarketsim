# Fee Market Simulator

A comprehensive **multi-algorithm** fee market simulator supporting various fee adjustment mechanisms including **EIP-1559**, **AIMD (Additive Increase Multiplicative Decrease)**, and **PID Controllers**. This simulator provides advanced features including burst capacity, randomness injection, real blockchain data integration, and visualization capabilities for comparing different fee adjustment strategies.

## 📦 Project Overview

```
├── cmd/simulator/          # Application entry point
├── pkg/
│   ├── config/             # Configuration management
│   ├── simulator/          # Core fee adjustment algorithm implementations
│   ├── scenarios/          # Simulation scenario generation
│   ├── analysis/           # Statistical analysis and reporting
│   ├── blockchain/         # Real blockchain data integration
│   ├── randomizer/         # Data randomization
|   └── visualization/      # Chart generation
|   
└── go.mod
```

## 🔧 Supported Algorithms

### 1. EIP-1559 (Standard Ethereum)

The standard Ethereum fee adjustment mechanism as specified in EIP-1559.

#### Algorithm

```
gasUsedDelta = gasUsed - targetGas
baseFeeChange = baseFee * gasUsedDelta / targetGas / 8
newBaseFee = baseFee + baseFeeChange
```

#### EIP-1559 Configuration Parameters

| Parameter | Description | Default Value |
|-----------|-------------|---------------|
| `MaxFeeChange` | Maximum fee change per block | 0.125 (12.5%) |

### 2. AIMD (Additive Increase Multiplicative Decrease)

An enhanced variant of EIP-1559 with adaptive learning rates and historical window analysis.

#### Core Components

1. **Dynamic Block Capacity**:
   ```
   maxBlockSize = targetBlockSize * burstMultiplier
   ```

2. **Target Utilization Calculation**:
   ```
   targetUtilization = sumBlockSizesInWindow(window) / (window * targetBlockSize)
   ```

3. **Learning Rate Adjustment**:
   ```
   utilizationDeviation = |targetUtilization - 1.0|
   if utilizationDeviation > gamma:
       newLearningRate = min(MaxLearningRate, α + currentLearningRate)
   else:
       newLearningRate = max(MinLearningRate, β * currentLearningRate)
   ```

4. **Base Fee Update**:
   ```
   newBaseFee = currentBaseFee * (1 + learningRate * (currentBlockSize - targetBlockSize) / targetBlockSize) + δ * netGasDelta(window)
   ```

#### AIMD Configuration Parameters

| Parameter | Description | Default Value |
|-----------|-------------|---------------|
| `WindowSize` | Blocks in analysis window | 10 blocks |
| `Gamma (γ)` | Target utilization deviation threshold | 0.25 (25%) |
| `MaxLearningRate` | Maximum learning rate | 0.5 (50%) |
| `MinLearningRate` | Minimum learning rate | 0.001 (0.1%) |
| `Alpha (α)` | Additive increase factor | 0.01 (1%) |
| `Beta (β)` | Multiplicative decrease factor | 0.9 (90%) |
| `Delta (δ)` | Net gas delta coefficient | 0.000001 |
| `InitialLearningRate` | Initial learning rate | 0.1 (10%) |

### 3. PID Controller

A Proportional-Integral-Derivative control system approach to fee adjustment.

#### Algorithm

```
error = (gasUsed / targetBlockSize) - 1.0
proportional = Kp * error
integral += error (with windup protection)
derivative = slope of recent errors
controlOutput = proportional + Ki * integral + Kd * derivative
newBaseFee = baseFee * (1 + controlOutput)
```

#### PID Configuration Parameters

| Parameter | Description | Default Value |
|-----------|-------------|---------------|
| `Kp` | Proportional gain | 0.02 |
| `Ki` | Integral gain | 0.00001 |
| `Kd` | Derivative gain | 0.01 |
| `MaxIntegral` | Maximum integral value | 100.0 |
| `MinIntegral` | Minimum integral value | -100.0 |
| `MaxFeeChange` | Maximum fee change per block | 0.25 (25%) |
| `WindowSize` | Window for derivative calculation | 10 blocks |

### Common Configuration Parameters

| Parameter | Description | Default Value |
|-----------|-------------|---------------|
| `TargetBlockSize` | Target gas usage per block | 15M gas |
| `BurstMultiplier` | Max capacity as multiple of target | 2.0 (30M gas max) |
| `InitialBaseFee` | Initial base fee | 1 Gwei |
| `MinBaseFee` | Minimum base fee | 0 |
| `RandomnessFactor` | Gaussian noise level | 0.1 (10%) |

## 🚀 Usage Guide

### Installation

```bash
# Clone the repository
git clone https://github.com/brianbland/feemarketsim
cd feemarketsim

# Build the simulator
go build -o feemarketsim cmd/simulator/main.go

# Or run directly
go run cmd/simulator/main.go
```

### Basic Algorithm Comparison

```bash
# Compare all algorithms with default settings
./feemarketsim -adjuster-type=aimd -scenario=mixed -graph
./feemarketsim -adjuster-type=eip1559 -scenario=mixed -graph
./feemarketsim -adjuster-type=pid -scenario=mixed -graph

# Quick start with different algorithms
./feemarketsim -adjuster-type=aimd      # AIMD with adaptive learning
./feemarketsim -adjuster-type=eip1559   # Standard Ethereum mechanism
./feemarketsim -adjuster-type=pid       # PID controller approach
```

### Advanced Algorithm Configuration

#### AIMD Parameter Tuning
```bash
# Conservative AIMD (stable fees, slow adaptation)
./feemarketsim -adjuster-type=aimd -aimd-gamma=0.5 -aimd-alpha=0.005 -window-size=20

# Aggressive AIMD (responsive fees, fast adaptation)
./feemarketsim -adjuster-type=aimd -aimd-gamma=0.1 -aimd-alpha=0.02 -window-size=5

# Custom learning rate range
./feemarketsim -adjuster-type=aimd -aimd-min-learning-rate=0.0001 -aimd-max-learning-rate=0.8
```

#### PID Controller Tuning
```bash
# More aggressive proportional response
./feemarketsim -adjuster-type=pid -pid-kp=0.2

# Higher integral gain for steady-state accuracy
./feemarketsim -adjuster-type=pid -pid-ki=0.05

# More derivative action for faster response
./feemarketsim -adjuster-type=pid -pid-kd=0.1

# Complete PID tuning
./feemarketsim -adjuster-type=pid -pid-kp=0.15 -pid-ki=0.02 -pid-kd=0.08 -pid-max-fee-change=0.3
```

#### EIP-1559 Configuration
```bash
# More aggressive fee changes
./feemarketsim -adjuster-type=eip1559 -eip1559-max-fee-change=0.2

# Conservative fee changes
./feemarketsim -adjuster-type=eip1559 -eip1559-max-fee-change=0.1
```

### Real Blockchain Data Analysis

#### 1. Fetch Base Blockchain Data

```bash
# Fetch small range for testing
./feemarketsim fetch-base 12000000 12000010 test_data.json

# Fetch larger dataset (with confirmation prompt)
./feemarketsim fetch-base 12000000 12001000 base_data.json

# Fetch recent data
./feemarketsim fetch-base 18000000 18000500 recent_base.json
```

**Features:**
- Concurrent fetching with configurable worker pools
- Exponential backoff retry with jitter protection
- Gap detection ensuring complete datasets
- Progress reporting with real-time statistics
- Large range warnings with user confirmation

#### 2. Compare Algorithms Against Real Data

```bash
# Test all algorithms against the same dataset
./feemarketsim simulate-base base_data.json -adjuster-type=aimd -graph
./feemarketsim simulate-base base_data.json -adjuster-type=eip1559 -graph
./feemarketsim simulate-base base_data.json -adjuster-type=pid -graph

# With custom parameters and logarithmic scale
./feemarketsim simulate-base base_data.json -adjuster-type=aimd -aimd-gamma=0.1 -graph -log-scale
./feemarketsim simulate-base base_data.json -adjuster-type=pid -pid-kp=0.15 -graph -log-scale
```

### Complete Command Reference

#### Algorithm Selection
```bash
-adjuster-type=aimd             # AIMD - Adaptive learning rate algorithm
-adjuster-type=eip1559          # EIP-1559 - Standard Ethereum mechanism
-adjuster-type=pid              # PID Controller - Industrial control system
```

#### Core Parameters (apply to all algorithms)
```bash
-target-block-size=15000000     # Target block size in gas units
-burst-multiplier=2.0           # Max burst capacity multiplier
-initial-base-fee=1000000000    # Initial base fee in wei
-min-base-fee=0                 # Minimum base fee in wei
```

#### AIMD-Specific Parameters
```bash
-window-size=10                 # Analysis window size in blocks
-aimd-gamma=0.25                # Learning rate adjustment threshold
-aimd-max-learning-rate=0.5     # Maximum learning rate
-aimd-min-learning-rate=0.001   # Minimum learning rate
-aimd-alpha=0.01                # Additive increase factor
-aimd-beta=0.9                  # Multiplicative decrease factor
-aimd-delta=0.000001            # Net gas delta coefficient
-aimd-initial-learning-rate=0.1 # Initial learning rate
```

#### EIP-1559 Parameters
```bash
-eip1559-max-fee-change=0.125   # Maximum fee change per block
```

#### PID Controller Parameters
```bash
-window-size=10                 # Window for derivative calculation
-pid-kp=0.02                    # Proportional gain
-pid-ki=0.00001                 # Integral gain
-pid-kd=0.01                    # Derivative gain
-pid-max-integral=100.0         # Maximum integral value
-pid-min-integral=-100.0        # Minimum integral value
-pid-max-fee-change=0.25        # Maximum fee change per block
```

#### Simulation Control
```bash
-scenario=all                   # Scenario selection (full, empty, stable, mixed, all)
-graph                          # Generate visualization charts
-log-scale                      # Use logarithmic scale for Y-axis in charts
-help                           # Show detailed help
```

## 📊 Simulation Scenarios

### 1. **Extended Full Blocks** (35 blocks)
- Sustained high congestion testing
- 180-200% of target capacity
- Tests aggressive fee increases across all algorithms

### 2. **Extended Empty Blocks** (35 blocks)
- Sustained low demand testing
- 2-13% of target capacity
- Tests fee reduction mechanisms and algorithm stability

### 3. **Extended Stable** (40 blocks)
- Long-term stability testing
- 85-115% of target capacity
- Tests different algorithms' ability to maintain steady fees

### 4. **Extended Mixed Traffic** (80 blocks)
- Realistic traffic patterns
- Gradual transitions between states
- Tests adaptability and responsiveness of each algorithm

### Algorithm Performance Comparison

Each scenario can be run with different algorithms to compare:
- **AIMD**: Adaptive learning rate behavior and window-based analysis
- **EIP-1559**: Standard Ethereum baseline performance
- **PID**: Control system stability and response characteristics

### Chart Output
Generated files include:
- `chart_[algorithm]_[scenario]_[params].html` - Individual algorithm analysis
- `comparison_[scenario]_[algorithms].html` - Multi-algorithm comparison
- `base_comparison_[start]_[end].html` - Real data comparison
- `base_comparison_[start]_[end]_gas.html` - Gas usage analysis

## 🔬 Algorithm Comparison Examples

### Quick Algorithm Comparison
```bash
# Compare all algorithms on the same scenario
./feemarketsim -adjuster-type=aimd -scenario=mixed -graph
./feemarketsim -adjuster-type=eip1559 -scenario=mixed -graph  
./feemarketsim -adjuster-type=pid -scenario=mixed -graph
```

### Parameter Sensitivity Analysis
```bash
# AIMD parameter testing
./feemarketsim -adjuster-type=aimd -aimd-gamma=0.1 -aimd-alpha=0.02    # Aggressive
./feemarketsim -adjuster-type=aimd -aimd-gamma=0.5 -aimd-alpha=0.005   # Conservative

# PID controller tuning  
./feemarketsim -adjuster-type=pid -pid-kp=0.2                          # More aggressive P
./feemarketsim -adjuster-type=pid -pid-ki=0.05                         # Higher integral gain

# EIP-1559 variants
./feemarketsim -adjuster-type=eip1559 -eip1559-max-fee-change=0.2      # More aggressive
./feemarketsim -adjuster-type=eip1559 -eip1559-max-fee-change=0.08     # More conservative
```

### Real Blockchain Data Comparison
```bash
# 1. Fetch blockchain data
./feemarketsim fetch-base 12000000 12001000 analysis.json

# 2. Test different algorithms on the same data
./feemarketsim simulate-base analysis.json -adjuster-type=aimd -graph
./feemarketsim simulate-base analysis.json -adjuster-type=eip1559 -graph
./feemarketsim simulate-base analysis.json -adjuster-type=pid -graph

# 3. Compare with parameter variations
./feemarketsim simulate-base analysis.json -adjuster-type=aimd -aimd-gamma=0.1 -graph
./feemarketsim simulate-base analysis.json -adjuster-type=pid -pid-kp=0.15 -graph
```

### Advanced Analysis Workflows

#### Stability Analysis
```bash
# Test algorithm stability under different conditions
./feemarketsim -adjuster-type=aimd -scenario=stable -window-size=20
./feemarketsim -adjuster-type=eip1559 -scenario=stable
./feemarketsim -adjuster-type=pid -scenario=stable -pid-ki=0.001  # Low integral gain
```

#### Responsiveness Testing
```bash
# Test response to sudden changes
./feemarketsim -adjuster-type=aimd -scenario=mixed -aimd-alpha=0.03
./feemarketsim -adjuster-type=pid -scenario=mixed -pid-kp=0.25
./feemarketsim -adjuster-type=eip1559 -scenario=mixed -eip1559-max-fee-change=0.25
```

#### Burst Capacity Analysis
```bash
# Test different burst configurations across algorithms
./feemarketsim -adjuster-type=aimd -burst-multiplier=3.0 -aimd-gamma=0.1
./feemarketsim -adjuster-type=eip1559 -burst-multiplier=3.0
./feemarketsim -adjuster-type=pid -burst-multiplier=3.0 -pid-max-fee-change=0.3
```

### Running Tests
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/simulator/
go test ./pkg/blockchain/
go test ./pkg/visualization/

# Test specific algorithms
go test ./pkg/simulator/ -run TestAdjusterTypes
go test ./pkg/simulator/ -run TestFactoryWithConfigs
```

## 📈 Performance Characteristics

### AIMD Algorithm
- **Strengths**: Adaptive learning, window-based analysis, handles sustained congestion well
- **Use Cases**: Networks with variable traffic patterns, when fine-tuned responsiveness is needed
- **Tuning**: Adjust gamma for responsiveness, alpha/beta for learning rate behavior

### EIP-1559 
- **Strengths**: Simple, predictable, battle-tested on Ethereum mainnet
- **Use Cases**: Baseline comparison, production environments requiring proven stability
- **Tuning**: Limited to max fee change parameter

### PID Controller
- **Strengths**: Control system theory foundations, good steady-state accuracy, configurable response
- **Use Cases**: When precise fee targeting is needed, systems requiring minimal overshoot
- **Tuning**: Classic PID tuning methods apply (Ziegler-Nichols, etc.)

## 🛠️ Development and Contribution

### Adding New Algorithms

To add a new fee adjustment algorithm:

1. **Implement the `FeeAdjuster` interface** in `pkg/simulator/`
2. **Add configuration struct** following existing patterns
3. **Update the factory** in `pkg/simulator/factory.go`
4. **Add CLI flags** in `pkg/config/config.go`
5. **Add tests** in `pkg/simulator/`

### Testing New Algorithms

```bash
# Test against all scenarios
./feemarketsim -adjuster-type=your-algorithm -scenario=all -graph

# Compare against existing algorithms
./feemarketsim -adjuster-type=your-algorithm -scenario=mixed -graph
./feemarketsim -adjuster-type=aimd -scenario=mixed -graph
./feemarketsim -adjuster-type=eip1559 -scenario=mixed -graph

# Test with real data
./feemarketsim simulate-base your_data.json -adjuster-type=your-algorithm -graph
```