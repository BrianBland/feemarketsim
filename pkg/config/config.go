package config

import (
	"flag"
	"fmt"
	"time"
)

// Config holds the configuration parameters for the fee adjustment mechanism
type Config struct {
	// Core parameters (apply to all algorithms)
	TargetBlockSize uint64  // Target block size in gas units
	BurstMultiplier float64 // Max burst capacity as multiple of target (e.g., 2.0 = 200% of target)
	InitialBaseFee  uint64  // Initial base fee in wei
	MinBaseFee      uint64  // Minimum base fee in wei (default: 0)
	WindowSize      int     // Number of blocks to consider in the window
	Simulation      SimulationConfig
	Adjuster        AdjusterConfigs
}

// SimulationConfig holds runtime configuration for simulations
type SimulationConfig struct {
	Scenario     string
	EnableGraphs bool
	LogScale     bool // Use logarithmic scale for Y-axis in charts
	ShowHelp     bool
	AdjusterType string // Type of fee adjuster to use
	Randomizer   RandomizerConfig
}

// RandomizerConfig holds configuration for randomizer
type RandomizerConfig struct {
	Seed             int64   // Seed for randomizer
	GaussianNoise    float64 // Standard deviation for Gaussian noise (0.0 = none, 0.1 = 10% variation)
	BurstProbability float64 // Probability of entering burst mode each block (0.0 = none, 0.1 = 10%)
	BurstDurationMin int     // Minimum burst duration (blocks)
	BurstDurationMax int     // Maximum burst duration (blocks)
	BurstIntensity   float64 // Multiplier for gas usage during bursts
}

// AdjusterConfigs holds configuration for different adjuster types
type AdjusterConfigs struct {
	// EIP-1559 specific config
	EIP1559 struct {
		MaxFeeChange float64 // Maximum fee change per block (1/8 = 0.125)
	}

	// AIMD specific config
	AIMD struct {
		Gamma               float64 // Threshold for learning rate adjustment (relative to target utilization)
		MaxLearningRate     float64 // Maximum learning rate
		MinLearningRate     float64 // Minimum learning rate
		Alpha               float64 // Additive increase factor
		Beta                float64 // Multiplicative decrease factor
		Delta               float64 // Net gas delta coefficient
		InitialLearningRate float64 // Initial learning rate
	}

	// PID controller specific config
	PID struct {
		Kp           float64 // Proportional gain
		Ki           float64 // Integral gain
		Kd           float64 // Derivative gain
		MaxIntegral  float64 // Maximum integral value
		MinIntegral  float64 // Minimum integral value
		MaxFeeChange float64 // Maximum fee change per block
	}
}

// Default returns a configuration with sensible defaults
func Default() Config {
	cfg := Config{
		TargetBlockSize: 15_000_000,
		BurstMultiplier: 2.0,
		InitialBaseFee:  1_000_000_000,
		MinBaseFee:      0,
		WindowSize:      10,
		Simulation: SimulationConfig{
			Scenario:     "all",
			EnableGraphs: false,
			LogScale:     false,
			ShowHelp:     false,
			AdjusterType: "aimd",
			Randomizer: RandomizerConfig{
				Seed: time.Now().UnixNano(),
			},
		},
	}

	cfg.Adjuster.EIP1559.MaxFeeChange = 0.125

	cfg.Adjuster.AIMD.Gamma = 0.25
	cfg.Adjuster.AIMD.MaxLearningRate = 0.5
	cfg.Adjuster.AIMD.MinLearningRate = 0.001
	cfg.Adjuster.AIMD.Alpha = 0.01
	cfg.Adjuster.AIMD.Beta = 0.9
	cfg.Adjuster.AIMD.Delta = 0
	cfg.Adjuster.AIMD.InitialLearningRate = 0.1

	cfg.Adjuster.PID.Kp = 0.02
	cfg.Adjuster.PID.Ki = 0.00001
	cfg.Adjuster.PID.Kd = 0.01
	cfg.Adjuster.PID.MaxIntegral = 100.0
	cfg.Adjuster.PID.MinIntegral = -100.0
	cfg.Adjuster.PID.MaxFeeChange = 0.25

	return cfg
}

// Parser handles command-line flag parsing
type Parser struct {
	config  *Config
	flagSet *flag.FlagSet
}

// NewParser creates a new configuration parser
func NewParser() *Parser {
	config := Default()

	flagSet := flag.NewFlagSet("feemarketsim", flag.ExitOnError)

	return &Parser{
		config:  &config,
		flagSet: flagSet,
	}
}

// RegisterFlags registers all command-line flags
func (p *Parser) RegisterFlags() {
	// Core configuration flags (apply to all algorithms)
	p.flagSet.Uint64Var(&p.config.TargetBlockSize, "target-block-size", p.config.TargetBlockSize, "Target block size in gas units")
	p.flagSet.Float64Var(&p.config.BurstMultiplier, "burst-multiplier", p.config.BurstMultiplier, "Max burst capacity as multiple of target")
	p.flagSet.Uint64Var(&p.config.InitialBaseFee, "initial-base-fee", p.config.InitialBaseFee, "Initial base fee in wei")
	p.flagSet.Uint64Var(&p.config.MinBaseFee, "min-base-fee", p.config.MinBaseFee, "Minimum base fee in wei")

	// Simulation configuration flags
	p.flagSet.StringVar(&p.config.Simulation.Scenario, "scenario", p.config.Simulation.Scenario, "Scenario to run: full, empty, stable, mixed, or all")
	p.flagSet.BoolVar(&p.config.Simulation.EnableGraphs, "graph", p.config.Simulation.EnableGraphs, "Generate visualization charts (HTML files)")
	p.flagSet.BoolVar(&p.config.Simulation.LogScale, "log-scale", p.config.Simulation.LogScale, "Use logarithmic scale for Y-axis in charts")
	p.flagSet.BoolVar(&p.config.Simulation.ShowHelp, "help", p.config.Simulation.ShowHelp, "Show detailed help and parameter explanations")

	// Randomizer configuration flags
	p.flagSet.Int64Var(&p.config.Simulation.Randomizer.Seed, "rng-seed", p.config.Simulation.Randomizer.Seed, "Seed for randomizer")
	p.flagSet.Float64Var(&p.config.Simulation.Randomizer.GaussianNoise, "rng-gaussian-noise", p.config.Simulation.Randomizer.GaussianNoise, "Standard deviation for Gaussian noise (0.0 = none, 0.1 = 10% variation)")
	p.flagSet.Float64Var(&p.config.Simulation.Randomizer.BurstProbability, "rng-burst-probability", p.config.Simulation.Randomizer.BurstProbability, "Probability of entering burst mode per block (0.0 = none, 0.1 = 10%)")
	p.flagSet.IntVar(&p.config.Simulation.Randomizer.BurstDurationMin, "rng-burst-duration-min", p.config.Simulation.Randomizer.BurstDurationMin, "Minimum burst duration in blocks")
	p.flagSet.IntVar(&p.config.Simulation.Randomizer.BurstDurationMax, "rng-burst-duration-max", p.config.Simulation.Randomizer.BurstDurationMax, "Maximum burst duration in blocks")
	p.flagSet.Float64Var(&p.config.Simulation.Randomizer.BurstIntensity, "rng-burst-intensity", p.config.Simulation.Randomizer.BurstIntensity, "Multiplier for gas usage during bursts")

	// Common controller flags
	p.flagSet.IntVar(&p.config.WindowSize, "window-size", p.config.WindowSize, "Number of blocks to consider in the window")

	// Adjuster type flags
	p.flagSet.StringVar(&p.config.Simulation.AdjusterType, "adjuster-type", p.config.Simulation.AdjusterType, "Type of fee adjuster to use: aimd, eip1559, pid")

	// EIP-1559 specific flags
	p.flagSet.Float64Var(&p.config.Adjuster.EIP1559.MaxFeeChange, "eip1559-max-fee-change", p.config.Adjuster.EIP1559.MaxFeeChange, "EIP-1559: Maximum fee change per block")

	// AIMD controller specific flags
	p.flagSet.Float64Var(&p.config.Adjuster.AIMD.Gamma, "aimd-gamma", p.config.Adjuster.AIMD.Gamma, "AIMD: Threshold for learning rate adjustment")
	p.flagSet.Float64Var(&p.config.Adjuster.AIMD.MaxLearningRate, "aimd-max-learning-rate", p.config.Adjuster.AIMD.MaxLearningRate, "AIMD: Maximum learning rate")
	p.flagSet.Float64Var(&p.config.Adjuster.AIMD.MinLearningRate, "aimd-min-learning-rate", p.config.Adjuster.AIMD.MinLearningRate, "AIMD: Minimum learning rate")
	p.flagSet.Float64Var(&p.config.Adjuster.AIMD.Alpha, "aimd-alpha", p.config.Adjuster.AIMD.Alpha, "AIMD: Additive increase factor")
	p.flagSet.Float64Var(&p.config.Adjuster.AIMD.Beta, "aimd-beta", p.config.Adjuster.AIMD.Beta, "AIMD: Multiplicative decrease factor")
	p.flagSet.Float64Var(&p.config.Adjuster.AIMD.Delta, "aimd-delta", p.config.Adjuster.AIMD.Delta, "AIMD: Net gas delta coefficient")
	p.flagSet.Float64Var(&p.config.Adjuster.AIMD.InitialLearningRate, "aimd-initial-learning-rate", p.config.Adjuster.AIMD.InitialLearningRate, "AIMD: Initial learning rate")

	// PID controller specific flags
	p.flagSet.Float64Var(&p.config.Adjuster.PID.Kp, "pid-kp", p.config.Adjuster.PID.Kp, "PID: Proportional gain")
	p.flagSet.Float64Var(&p.config.Adjuster.PID.Ki, "pid-ki", p.config.Adjuster.PID.Ki, "PID: Integral gain")
	p.flagSet.Float64Var(&p.config.Adjuster.PID.Kd, "pid-kd", p.config.Adjuster.PID.Kd, "PID: Derivative gain")
	p.flagSet.Float64Var(&p.config.Adjuster.PID.MaxIntegral, "pid-max-integral", p.config.Adjuster.PID.MaxIntegral, "PID: Maximum integral value")
	p.flagSet.Float64Var(&p.config.Adjuster.PID.MinIntegral, "pid-min-integral", p.config.Adjuster.PID.MinIntegral, "PID: Minimum integral value")
	p.flagSet.Float64Var(&p.config.Adjuster.PID.MaxFeeChange, "pid-max-fee-change", p.config.Adjuster.PID.MaxFeeChange, "PID: Maximum fee change per block")
}

// Parse parses command-line arguments and returns configuration
func (p *Parser) Parse(args []string) (*Config, error) {
	p.RegisterFlags()

	if err := p.flagSet.Parse(args); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	if p.config.Simulation.ShowHelp {
		p.ShowDetailedHelp()
		return p.config, nil
	}

	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return p.config, nil
}

// Validate validates the configuration parameters
func (p *Parser) Validate() error {
	c := p.config
	s := &p.config.Simulation
	a := &p.config.Adjuster

	// Validate adjuster type
	validAdjusterTypes := []string{"aimd", "eip1559", "eip-1559", "pid"}
	isValidAdjusterType := false
	for _, validType := range validAdjusterTypes {
		if s.AdjusterType == validType {
			isValidAdjusterType = true
			break
		}
	}
	if !isValidAdjusterType {
		return fmt.Errorf("invalid adjuster type '%s', must be one of: %v", s.AdjusterType, validAdjusterTypes)
	}

	// Core parameter validation (applies to all algorithms)
	if c.BurstMultiplier <= 1.0 {
		return fmt.Errorf("burst multiplier (%.3f) must be greater than 1.0", c.BurstMultiplier)
	}

	// Randomizer validation
	if err := p.validateRandomizerParameters(s); err != nil {
		return err
	}

	// Algorithm-specific validation
	switch s.AdjusterType {
	case "aimd":
		// Validate AIMD-specific parameters
		if err := p.validateAIMDParameters(a); err != nil {
			return err
		}

	case "eip1559", "eip-1559":
		// Validate EIP-1559 parameters
		if a.EIP1559.MaxFeeChange <= 0 || a.EIP1559.MaxFeeChange > 1.0 {
			return fmt.Errorf("EIP-1559 max fee change (%.3f) must be between 0 and 1.0", a.EIP1559.MaxFeeChange)
		}

	case "pid":
		// Validate PID parameters
		if err := p.validatePIDParameters(a); err != nil {
			return err
		}
	}

	// Scenario validation
	validScenarios := []string{"all", "full", "empty", "stable", "mixed"}
	isValid := false
	for _, valid := range validScenarios {
		if s.Scenario == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid scenario '%s', must be one of: %v", s.Scenario, validScenarios)
	}

	return nil
}

// validateAIMDParameters validates AIMD-specific parameters
func (p *Parser) validateAIMDParameters(a *AdjusterConfigs) error {
	if a.AIMD.Gamma < 0 || a.AIMD.Gamma > 2.0 {
		return fmt.Errorf("gamma (%.3f) must be between 0 and 2.0", a.AIMD.Gamma)
	}

	if a.AIMD.MaxLearningRate < a.AIMD.MinLearningRate {
		return fmt.Errorf("max learning rate (%.6f) must be >= min learning rate (%.6f)",
			a.AIMD.MaxLearningRate, a.AIMD.MinLearningRate)
	}

	if a.AIMD.Alpha < 0 {
		return fmt.Errorf("alpha (%.6f) must not be negative", a.AIMD.Alpha)
	}

	if a.AIMD.Beta < 0 || a.AIMD.Beta > 1 {
		return fmt.Errorf("beta (%.6f) must be between 0 and 1", a.AIMD.Beta)
	}

	if p.config.WindowSize <= 0 {
		return fmt.Errorf("window size (%d) must be positive", p.config.WindowSize)
	}

	return nil
}

// validatePIDParameters validates PID-specific parameters
func (p *Parser) validatePIDParameters(a *AdjusterConfigs) error {
	if a.PID.Kp < 0 {
		return fmt.Errorf("PID Kp (%.6f) must not be negative", a.PID.Kp)
	}
	if a.PID.Ki < 0 {
		return fmt.Errorf("PID Ki (%.6f) must not be negative", a.PID.Ki)
	}
	if a.PID.Kd < 0 {
		return fmt.Errorf("PID Kd (%.6f) must not be negative", a.PID.Kd)
	}
	if a.PID.MaxIntegral <= a.PID.MinIntegral {
		return fmt.Errorf("PID max integral (%.3f) must be greater than min integral (%.3f)", a.PID.MaxIntegral, a.PID.MinIntegral)
	}
	if a.PID.MaxFeeChange <= 0 || a.PID.MaxFeeChange > 1.0 {
		return fmt.Errorf("PID max fee change (%.3f) must be between 0 and 1.0", a.PID.MaxFeeChange)
	}
	if p.config.WindowSize <= 0 {
		return fmt.Errorf("PID window size (%d) must be positive", p.config.WindowSize)
	}
	return nil
}

// validateRandomizerParameters validates randomizer parameters
func (p *Parser) validateRandomizerParameters(a *SimulationConfig) error {
	if a.Randomizer.GaussianNoise < 0 || a.Randomizer.GaussianNoise > 1.0 {
		return fmt.Errorf("randomizer gaussian noise (%.3f) must be between 0.0 and 1.0", a.Randomizer.GaussianNoise)
	}
	if a.Randomizer.BurstProbability < 0 || a.Randomizer.BurstProbability > 1.0 {
		return fmt.Errorf("randomizer burst probability (%.3f) must be between 0.0 and 1.0", a.Randomizer.BurstProbability)
	}

	if a.Randomizer.BurstProbability > 0 {
		if a.Randomizer.BurstDurationMin <= 0 {
			return fmt.Errorf("randomizer burst duration min (%d) must be positive", a.Randomizer.BurstDurationMin)
		}
		if a.Randomizer.BurstDurationMax < a.Randomizer.BurstDurationMin {
			return fmt.Errorf("randomizer burst duration max (%d) must be >= min (%d)", a.Randomizer.BurstDurationMax, a.Randomizer.BurstDurationMin)
		}
		if a.Randomizer.BurstIntensity <= 0 {
			return fmt.Errorf("randomizer burst intensity (%.3f) must be positive", a.Randomizer.BurstIntensity)
		}
	}
	return nil
}

// ShowDetailedHelp displays comprehensive help information
func (p *Parser) ShowDetailedHelp() {
	fmt.Println("AIMD Fee Market Simulation - Complete CLI Reference")
	fmt.Println("================================================================================")
	fmt.Println()

	fmt.Println("OVERVIEW:")
	fmt.Println("  The AIMD Fee Market Simulator provides three main operation modes:")
	fmt.Println("  1. Basic AIMD Simulation - Test algorithm with synthetic scenarios")
	fmt.Println("  2. Base Blockchain Data Fetching - Download real blockchain data")
	fmt.Println("  3. Base Blockchain Simulation - Run AIMD against real data")
	fmt.Println()

	fmt.Println("COMMANDS:")
	fmt.Println()

	fmt.Println("Basic AIMD Simulation:")
	fmt.Println("  feemarketsim [flags]                   # Run with synthetic scenarios")
	fmt.Println("  feemarketsim -scenario=full -graph     # Run specific scenario with charts")
	fmt.Println("  feemarketsim -help                     # Show this help")
	fmt.Println()

	fmt.Println("Base Blockchain Integration:")
	fmt.Println("  feemarketsim fetch-base <start> <end> <file>  # Fetch blockchain data")
	fmt.Println("    - Example: feemarketsim fetch-base 12000000 12000100 data.json")
	fmt.Println("    - Downloads real Base blockchain data for analysis")
	fmt.Println("    - Supports concurrent fetching with retry logic")
	fmt.Println("    - Warns for large ranges (>10,000 blocks)")
	fmt.Println()
	fmt.Println("  feemarketsim simulate-base <file> [flags]     # Simulate against real data")
	fmt.Println("    - Example: feemarketsim simulate-base data.json -graph -gamma=0.1")
	fmt.Println("    - Runs fee adjustment algorithm against fetched blockchain data")
	fmt.Println("    - Supports all relevant parameter flags based on selected algorithm")
	fmt.Println("    - Generates comparison charts with -graph")
	fmt.Println()

	fmt.Println("ALGORITHM SELECTION:")
	fmt.Println()
	fmt.Println("  -adjuster-type=aimd          # AIMD (default) - Adaptive algorithm with learning")
	fmt.Println("  -adjuster-type=eip1559       # EIP-1559 - Standard Ethereum mechanism")
	fmt.Println("  -adjuster-type=pid           # PID Controller - Industrial control system")
	fmt.Println()

	fmt.Println("CORE PARAMETERS (apply to all algorithms):")
	fmt.Println()

	fmt.Println("Block Configuration:")
	fmt.Println("  -target-block-size=15000000  Target block size in gas units")
	fmt.Printf("                               Default: %d (%.1f M gas)\n", p.config.TargetBlockSize, float64(p.config.TargetBlockSize)/1e6)
	fmt.Println("  -burst-multiplier=2.0        Max burst capacity as multiple of target")
	fmt.Printf("                               Default: %.1f (%.1fx target capacity)\n", p.config.BurstMultiplier, p.config.BurstMultiplier)
	fmt.Println()

	fmt.Println("Fee Market Configuration:")
	fmt.Println("  -initial-base-fee=1000000000 Initial base fee in wei")
	fmt.Printf("                               Default: %d wei (%.3f Gwei)\n", p.config.InitialBaseFee, float64(p.config.InitialBaseFee)/1e9)
	fmt.Println("  -min-base-fee=0              Minimum base fee in wei")
	fmt.Printf("                               Default: %d wei (%.3f Gwei)\n", p.config.MinBaseFee, float64(p.config.MinBaseFee)/1e9)
	fmt.Println()

	fmt.Println("AIMD-SPECIFIC PARAMETERS (only for -adjuster-type=aimd or aimd-eip1559):")
	fmt.Println()

	fmt.Println("Learning System:")
	fmt.Println("  -window-size=10              Number of blocks in analysis window (only appies to AIMD and PID)")
	fmt.Printf("                               Default: %d blocks\n", p.config.WindowSize)
	fmt.Println()

	fmt.Println("AIMD Control:")
	fmt.Println("  -aimd-alpha=0.01                  Additive increase factor")
	fmt.Printf("                               Default: %.6f (learning rate increase)\n", p.config.Adjuster.AIMD.Alpha)
	fmt.Println("  -aimd-beta=0.9                    Multiplicative decrease factor")
	fmt.Printf("                               Default: %.3f (learning rate decay)\n", p.config.Adjuster.AIMD.Beta)
	fmt.Println("  -aimd-delta=0.000001              Net gas delta coefficient")
	fmt.Printf("                               Default: %.9f (window gas delta impact)\n", p.config.Adjuster.AIMD.Delta)
	fmt.Println("  -aimd-gamma=0.2                   Threshold for learning rate adjustment")
	fmt.Printf("                               Default: %.3f (deviation from target utilization)\n", p.config.Adjuster.AIMD.Gamma)
	fmt.Println("  -aimd-initial-learning-rate=0.1   Initial learning rate")
	fmt.Printf("                               Default: %.3f (%.1f%% initial adjustment)\n", p.config.Adjuster.AIMD.InitialLearningRate, p.config.Adjuster.AIMD.InitialLearningRate*100)
	fmt.Println("  -aimd-max-learning-rate=0.5       Maximum learning rate")
	fmt.Printf("                               Default: %.3f (%.1f%% maximum adjustment)\n", p.config.Adjuster.AIMD.MaxLearningRate, p.config.Adjuster.AIMD.MaxLearningRate*100)
	fmt.Println("  -aimd-min-learning-rate=0.001     Minimum learning rate")
	fmt.Printf("                               Default: %.6f (%.3f%% minimum adjustment)\n", p.config.Adjuster.AIMD.MinLearningRate, p.config.Adjuster.AIMD.MinLearningRate*100)
	fmt.Println()

	fmt.Println("EIP-1559 PARAMETERS (only for -adjuster-type=eip1559):")
	fmt.Println()
	fmt.Println("  -eip1559-max-fee-change=0.125  Maximum fee change per block")
	fmt.Printf("                                 Default: %.3f (%.1f%% max change)\n", p.config.Adjuster.EIP1559.MaxFeeChange, p.config.Adjuster.EIP1559.MaxFeeChange*100)
	fmt.Println()

	fmt.Println("PID CONTROLLER PARAMETERS (only for -adjuster-type=pid):")
	fmt.Println()
	fmt.Println("  -pid-kp=0.1                  Proportional gain")
	fmt.Printf("                               Default: %.3f\n", p.config.Adjuster.PID.Kp)
	fmt.Println("  -pid-ki=0.01                 Integral gain")
	fmt.Printf("                               Default: %.3f\n", p.config.Adjuster.PID.Ki)
	fmt.Println("  -pid-kd=0.05                 Derivative gain")
	fmt.Printf("                               Default: %.3f\n", p.config.Adjuster.PID.Kd)
	fmt.Println("  -pid-max-fee-change=0.25     Maximum fee change per block")
	fmt.Printf("                               Default: %.3f (%.1f%% max change)\n", p.config.Adjuster.PID.MaxFeeChange, p.config.Adjuster.PID.MaxFeeChange*100)
	fmt.Println("  -pid-max-integral=1000       Maximum integral value (windup protection)")
	fmt.Printf("                               Default: %.1f\n", p.config.Adjuster.PID.MaxIntegral)
	fmt.Println("  -pid-min-integral=-1000      Minimum integral value (windup protection)")
	fmt.Printf("                               Default: %.1f\n", p.config.Adjuster.PID.MinIntegral)
	fmt.Println()

	fmt.Println("SIMULATION CONTROL:")
	fmt.Println()
	fmt.Println("  -scenario=all                Scenario to run")
	fmt.Printf("                               Default: %s\n", p.config.Simulation.Scenario)
	fmt.Println("                               Options: full, empty, stable, mixed, all")
	fmt.Println("                               - full:   Sustained high congestion (35 blocks)")
	fmt.Println("                               - empty:  Sustained low demand (35 blocks)")
	fmt.Println("                               - stable: Long-term stability (40 blocks)")
	fmt.Println("                               - mixed:  Realistic traffic patterns (240 blocks)")
	fmt.Println("                               - all:    Run all scenarios sequentially")
	fmt.Println("  -graph                       Generate visualization charts (HTML files)")
	fmt.Println("                               Creates fee evolution and comparison charts")
	fmt.Println("  -log-scale                   Use logarithmic scale for Y-axis in charts")
	fmt.Println("                               Useful when fees span multiple orders of magnitude")
	fmt.Println()

	fmt.Println("RANDOMIZER PARAMETERS (only when -enable-rng is used):")
	fmt.Println()
	fmt.Println("  -rng-seed=1234567890           Seed for randomizer")
	fmt.Printf("                               Default: unix timestamp\n")
	fmt.Println("  -rng-gaussian-noise=0.1       Gaussian noise level (0.0-1.0)")
	fmt.Printf("                               Default: %.3f (%.1f%% variation)\n", p.config.Simulation.Randomizer.GaussianNoise, p.config.Simulation.Randomizer.GaussianNoise*100)
	fmt.Println("  -rng-burst-probability=0.1    Probability of burst mode per block")
	fmt.Printf("                               Default: %.2f (%.0f%% chance per block)\n", p.config.Simulation.Randomizer.BurstProbability, p.config.Simulation.Randomizer.BurstProbability*100)
	fmt.Println("  -rng-burst-duration-min=2      Minimum burst duration in blocks")
	fmt.Printf("                               Default: %d blocks\n", p.config.Simulation.Randomizer.BurstDurationMin)
	fmt.Println("  -rng-burst-duration-max=5      Maximum burst duration in blocks")
	fmt.Printf("                               Default: %d blocks\n", p.config.Simulation.Randomizer.BurstDurationMax)
	fmt.Println("  -rng-burst-intensity=1.5       Gas usage multiplier during bursts")
	fmt.Printf("                               Default: %.1f (%.0f%% of normal)\n", p.config.Simulation.Randomizer.BurstIntensity, p.config.Simulation.Randomizer.BurstIntensity*100)
	fmt.Println()

	fmt.Println("EXAMPLE WORKFLOWS:")
	fmt.Println()

	fmt.Println("Quick Start:")
	fmt.Println("  feemarketsim                           # Run AIMD with default settings")
	fmt.Println("  feemarketsim -adjuster-type=eip1559    # Use EIP-1559 algorithm")
	fmt.Println("  feemarketsim -adjuster-type=pid        # Use PID controller")
	fmt.Println("  feemarketsim -scenario=mixed -graph    # Test mixed traffic with charts")
	fmt.Println("  feemarketsim -help                     # Show this help")
	fmt.Println()

	fmt.Println("Algorithm Comparison:")
	fmt.Println("  # Compare different algorithms on same scenario")
	fmt.Println("  feemarketsim -adjuster-type=aimd -scenario=mixed -graph")
	fmt.Println("  feemarketsim -adjuster-type=eip1559 -scenario=mixed -graph")
	fmt.Println("  feemarketsim -adjuster-type=pid -scenario=mixed -graph")
	fmt.Println("  feemarketsim -adjuster-type=aimd-eip1559 -scenario=mixed -graph")
	fmt.Println()

	fmt.Println("AIMD Parameter Testing:")
	fmt.Println("  # Test learning strategies")
	fmt.Println("  feemarketsim -adjuster-type=aimd -aimd-gamma=0.1 -aimd-alpha=0.02    # Aggressive")
	fmt.Println("  feemarketsim -adjuster-type=aimd -aimd-gamma=0.5 -aimd-alpha=0.005   # Conservative")
	fmt.Println("  # Test window sizes")
	fmt.Println("  feemarketsim -adjuster-type=aimd -window-size=5                      # Fast response")
	fmt.Println("  feemarketsim -adjuster-type=aimd -window-size=20                     # Stable response")
	fmt.Println()

	fmt.Println("PID Controller Tuning:")
	fmt.Println("  # Test different PID gains")
	fmt.Println("  feemarketsim -adjuster-type=pid -pid-kp=0.2                # More aggressive P")
	fmt.Println("  feemarketsim -adjuster-type=pid -pid-ki=0.05               # Higher integral gain")
	fmt.Println("  feemarketsim -adjuster-type=pid -pid-kd=0.1                # More derivative action")
	fmt.Println()

	fmt.Println("Advanced Randomness:")
	fmt.Println("  # Test with RNG")
	fmt.Println("  feemarketsim -rng-gaussian-noise=0.1                                                                                  # 10% gas variation")
	fmt.Println("  feemarketsim -rng-burst-probability=0.1 -rng-burst-duration-min=2 -rng-burst-duration-max=5 -rng-burst-intensity=1.5  # Add burst periods")
	fmt.Println()

	fmt.Println("Real Data Analysis:")
	fmt.Println("  # 1. Fetch blockchain data")
	fmt.Println("  feemarketsim fetch-base 12000000 12001000 analysis.json")
	fmt.Println()
	fmt.Println("  # 2. Test different algorithms")
	fmt.Println("  feemarketsim simulate-base analysis.json -adjuster-type=aimd -graph")
	fmt.Println("  feemarketsim simulate-base analysis.json -adjuster-type=eip1559 -graph")
	fmt.Println("  feemarketsim simulate-base analysis.json -adjuster-type=pid -graph")
	fmt.Println()
	fmt.Println("  # 3. Fine-tune parameters")
	fmt.Println("  feemarketsim simulate-base analysis.json -adjuster-type=aimd -aimd-gamma=0.1 -graph")
	fmt.Println("  feemarketsim simulate-base analysis.json -adjuster-type=pid -pid-kp=0.15 -graph")
	fmt.Println()

	fmt.Println("OUTPUT FILES:")
	fmt.Println("  When -graph is enabled, the following files are generated:")
	fmt.Println("  - chart_<scenario>.html            Fee evolution charts")
	fmt.Println("  - base_comparison_<range>.html     Algorithm vs Base fee comparison")
	fmt.Println("  - base_comparison_<range>_gas.html Gas usage analysis")
	fmt.Println()

	fmt.Println("For detailed algorithm descriptions, see the project README.md file.")
}
