# Fee Market Simulator

A comprehensive implementation of an enhanced **Additive Increase Multiplicative Decrease (AIMD)** variant of the EIP-1559 fee market adjustment mechanism. This simulator provides advanced features including burst capacity, randomness injection, real blockchain data integration, and visualization capabilities.

## 📦 Project Overview

```
├── cmd/simulator/          # Application entry point
├── pkg/
│   ├── config/             # Configuration management
│   ├── simulator/          # Core AIMD algorithm implementation
│   ├── scenarios/          # Simulation scenario generation
│   ├── analysis/           # Statistical analysis and reporting
│   ├── blockchain/         # Real blockchain data integration
│   └── visualization/      # Chart generation
└── go.mod
```

### Package Responsibilities

- **`pkg/config`**: Centralized configuration with CLI parsing and validation
- **`pkg/simulator`**: Core AIMD fee adjustment algorithm with deterministic testing
- **`pkg/scenarios`**: Traffic pattern generation for comprehensive testing
- **`pkg/analysis`**: Statistical analysis with responsiveness scoring
- **`pkg/blockchain`**: Concurrent blockchain data fetching and simulation
- **`pkg/visualization`**: Chart generation with multiple output formats

## 🔧 Algorithm Description

### Enhanced Core Components

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

### EIP-1559 Compatibility Mode

When using the `-no-aimd` flag, the simulator switches to standard EIP-1559 behavior by setting:
- **Alpha (α) = 0**: No additive learning rate increases
- **Beta (β) = 1**: No multiplicative learning rate decreases
- **Gamma (γ) = 1**: No learning rate adjustments based on utilization deviation
- **Delta (δ) = 0**: No net gas delta from historical window
- **Window Size = 1**: Single block analysis (standard EIP-1559)

This provides a baseline for comparing the enhanced AIMD algorithm against the original EIP-1559 specification.

### Configuration Parameters

| Parameter | Description | Default Value |
|-----------|-------------|---------------|
| `TargetBlockSize` | Target gas usage per block | 15M gas |
| `BurstMultiplier` | Max capacity as multiple of target | 2.0 (30M gas max) |
| `WindowSize` | Blocks in analysis window | 10 blocks |
| `Gamma (γ)` | Target utilization deviation threshold | 0.2 (20%) |
| `MaxLearningRate` | Maximum learning rate | 0.5 (50%) |
| `MinLearningRate` | Minimum learning rate | 0.001 (0.1%) |
| `Alpha (α)` | Additive increase factor | 0.01 (1%) |
| `Beta (β)` | Multiplicative decrease factor | 0.9 (90%) |
| `Delta (δ)` | Net gas delta coefficient | 0.000001 |
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

### Basic AIMD Simulation

```bash
# Run all scenarios with default parameters
./feemarketsim

# Run specific scenario
./feemarketsim -scenario=mixed

# Generate visualization charts
./feemarketsim -scenario=stable -graph

# Generate charts with logarithmic Y-axis scaling
./feemarketsim -scenario=mixed -graph -log-scale

# Use standard EIP-1559 instead of AIMD
./feemarketsim -no-aimd

# Compare AIMD vs EIP-1559 behavior
./feemarketsim -scenario=mixed -graph                    # AIMD algorithm
./feemarketsim -no-aimd -scenario=mixed -graph           # EIP-1559 baseline
./feemarketsim -scenario=mixed -graph -log-scale         # AIMD with log scale

# Custom parameter testing
./feemarketsim -gamma=0.1 -alpha=0.02 -burst-multiplier=2.5

# High randomness testing
./feemarketsim -randomness=0.3 -scenario=mixed -graph
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

#### 2. Simulate Against Real Data

```bash
# Basic simulation against real data
./feemarketsim simulate-base base_data.json

# With custom parameters and visualization
./feemarketsim simulate-base base_data.json -graph -gamma=0.1 -alpha=0.02

# With logarithmic scale for large fee ranges
./feemarketsim simulate-base base_data.json -graph -log-scale

# Conservative vs aggressive comparison
./feemarketsim simulate-base base_data.json -gamma=0.5 -beta=0.95  # Conservative
./feemarketsim simulate-base base_data.json -gamma=0.1 -alpha=0.03  # Aggressive
```

### Advanced Configuration Examples

#### Burst Capacity Tuning
```bash
# Conservative burst (50% above target)
./feemarketsim -burst-multiplier=1.5 -gamma=0.3

# Aggressive burst (200% above target)
./feemarketsim -burst-multiplier=3.0 -gamma=0.1

# Ethereum-like configuration
./feemarketsim -burst-multiplier=2.0 -gamma=0.2
```

#### Learning Rate Strategies
```bash
# Stable fees (slow adaptation)
./feemarketsim -gamma=0.5 -alpha=0.005 -window-size=20

# Responsive fees (fast adaptation)
./feemarketsim -gamma=0.1 -alpha=0.02 -window-size=5

# Balanced approach
./feemarketsim -gamma=0.2 -alpha=0.01 -window-size=10
```

### Complete Command Reference

#### Core AIMD Parameters
```bash
-target-block-size=15000000     # Target block size in gas units
-burst-multiplier=2.0           # Max burst capacity multiplier
-window-size=10                 # Analysis window size in blocks
-gamma=0.2                      # Learning rate adjustment threshold
-max-learning-rate=0.5          # Maximum learning rate
-min-learning-rate=0.001        # Minimum learning rate
-alpha=0.01                     # Additive increase factor
-beta=0.9                       # Multiplicative decrease factor
-delta=0.000001                 # Net gas delta coefficient
```

#### Fee Configuration
```bash
-initial-base-fee=1000000000    # Initial base fee in wei
-min-base-fee=0                 # Minimum base fee in wei
-initial-learning-rate=0.1      # Initial learning rate
```

#### Simulation Control
```bash
-randomness=0.1                 # Gaussian noise level (0.0-1.0)
-scenario=all                   # Scenario selection
-graph                          # Generate visualization charts
-log-scale                      # Use logarithmic scale for Y-axis in charts
-no-aimd                        # Use standard EIP-1559 instead of AIMD
                                # Sets: alpha=0, beta=1, gamma=1, delta=0, window-size=1
                                # Cannot be combined with those individual flags
-help                           # Show detailed help
```

## 📊 Simulation Scenarios

### 1. **Extended Full Blocks** (35 blocks)
- Sustained high congestion testing
- 180-200% of target capacity
- Tests aggressive fee increases

### 2. **Extended Empty Blocks** (35 blocks)
- Sustained low demand testing
- 2-13% of target capacity
- Tests fee reduction mechanisms

### 3. **Extended Stable** (40 blocks)
- Long-term stability testing
- 85-115% of target capacity
- Tests multiplicative decrease behavior

### 4. **Extended Mixed Traffic** (80 blocks)
- Realistic traffic patterns
- Gradual transitions between states
- Tests adaptability and responsiveness

### Chart Output
Generated files include:
- `chart_[scenario]_[params].html` - AIMD scenario analysis
- `base_comparison_[start]_[end].html` - Main fee comparison
- `base_comparison_[start]_[end]_gas.html` - Gas usage analysis

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
```
