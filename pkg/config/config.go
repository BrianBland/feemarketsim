package config

import (
	"flag"
	"fmt"
)

// Config holds the configuration parameters for the fee adjustment mechanism
type Config struct {
	TargetBlockSize     uint64  // Target block size in gas units
	BurstMultiplier     float64 // Max burst capacity as multiple of target (e.g., 2.0 = 200% of target)
	WindowSize          int     // Number of blocks to consider in the window
	Gamma               float64 // Threshold for learning rate adjustment (relative to target utilization)
	MaxLearningRate     float64 // Maximum learning rate
	MinLearningRate     float64 // Minimum learning rate
	Alpha               float64 // Additive increase factor
	Beta                float64 // Multiplicative decrease factor
	Delta               float64 // Net gas delta coefficient
	InitialBaseFee      uint64  // Initial base fee in wei
	InitialLearningRate float64 // Initial learning rate
	RandomnessFactor    float64 // Amount of randomness to add (0.0 = none, 0.1 = 10% variation)
	MinBaseFee          uint64  // Minimum base fee in wei (default: 0)
}

// SimulationConfig holds runtime configuration for simulations
type SimulationConfig struct {
	Scenario     string
	EnableGraphs bool
	ShowHelp     bool
}

// Default returns a configuration with sensible defaults
func Default() Config {
	return Config{
		TargetBlockSize:     15_000_000,
		BurstMultiplier:     2.0,
		WindowSize:          10,
		Gamma:               0.2,
		MaxLearningRate:     0.5,
		MinLearningRate:     0.001,
		Alpha:               0.01,
		Beta:                0.9,
		Delta:               0.000001,
		InitialBaseFee:      1_000_000_000,
		InitialLearningRate: 0.1,
		RandomnessFactor:    0.1,
		MinBaseFee:          0,
	}
}

// Parser handles command-line flag parsing
type Parser struct {
	config    *Config
	simConfig *SimulationConfig
	flagSet   *flag.FlagSet
}

// NewParser creates a new configuration parser
func NewParser() *Parser {
	config := Default()
	simConfig := &SimulationConfig{
		Scenario:     "all",
		EnableGraphs: false,
		ShowHelp:     false,
	}

	flagSet := flag.NewFlagSet("aimd-simulator", flag.ExitOnError)

	return &Parser{
		config:    &config,
		simConfig: simConfig,
		flagSet:   flagSet,
	}
}

// RegisterFlags registers all command-line flags
func (p *Parser) RegisterFlags() {
	// Core configuration flags
	p.flagSet.Uint64Var(&p.config.TargetBlockSize, "target-block-size", p.config.TargetBlockSize, "Target block size in gas units")
	p.flagSet.Float64Var(&p.config.BurstMultiplier, "burst-multiplier", p.config.BurstMultiplier, "Max burst capacity as multiple of target")
	p.flagSet.IntVar(&p.config.WindowSize, "window-size", p.config.WindowSize, "Number of blocks in analysis window")
	p.flagSet.Float64Var(&p.config.Gamma, "gamma", p.config.Gamma, "Threshold for learning rate adjustment")
	p.flagSet.Float64Var(&p.config.MaxLearningRate, "max-learning-rate", p.config.MaxLearningRate, "Maximum learning rate")
	p.flagSet.Float64Var(&p.config.MinLearningRate, "min-learning-rate", p.config.MinLearningRate, "Minimum learning rate")
	p.flagSet.Float64Var(&p.config.Alpha, "alpha", p.config.Alpha, "Additive increase factor")
	p.flagSet.Float64Var(&p.config.Beta, "beta", p.config.Beta, "Multiplicative decrease factor")
	p.flagSet.Float64Var(&p.config.Delta, "delta", p.config.Delta, "Net gas delta coefficient")
	p.flagSet.Uint64Var(&p.config.InitialBaseFee, "initial-base-fee", p.config.InitialBaseFee, "Initial base fee in wei")
	p.flagSet.Float64Var(&p.config.InitialLearningRate, "initial-learning-rate", p.config.InitialLearningRate, "Initial learning rate")
	p.flagSet.Float64Var(&p.config.RandomnessFactor, "randomness", p.config.RandomnessFactor, "Amount of randomness to add (0.0-1.0)")
	p.flagSet.Uint64Var(&p.config.MinBaseFee, "min-base-fee", p.config.MinBaseFee, "Minimum base fee in wei")

	// Simulation configuration flags
	p.flagSet.StringVar(&p.simConfig.Scenario, "scenario", p.simConfig.Scenario, "Scenario to run: full, empty, stable, mixed, or all")
	p.flagSet.BoolVar(&p.simConfig.EnableGraphs, "graph", p.simConfig.EnableGraphs, "Generate visualization charts (PNG files)")
	p.flagSet.BoolVar(&p.simConfig.ShowHelp, "help", p.simConfig.ShowHelp, "Show detailed help and parameter explanations")
}

// Parse parses command-line arguments and returns configuration
func (p *Parser) Parse(args []string) (*Config, *SimulationConfig, error) {
	p.RegisterFlags()

	if err := p.flagSet.Parse(args); err != nil {
		return nil, nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	if p.simConfig.ShowHelp {
		p.ShowDetailedHelp()
		return p.config, p.simConfig, nil
	}

	if err := p.Validate(); err != nil {
		return nil, nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return p.config, p.simConfig, nil
}

// Validate validates the configuration parameters
func (p *Parser) Validate() error {
	c := p.config
	s := p.simConfig

	if c.BurstMultiplier <= 1.0 {
		return fmt.Errorf("burst multiplier (%.3f) must be greater than 1.0", c.BurstMultiplier)
	}

	if c.Gamma < 0 || c.Gamma > 2.0 {
		return fmt.Errorf("gamma (%.3f) must be between 0 and 2.0", c.Gamma)
	}

	if c.MaxLearningRate < c.MinLearningRate {
		return fmt.Errorf("max learning rate (%.6f) must be >= min learning rate (%.6f)",
			c.MaxLearningRate, c.MinLearningRate)
	}

	if c.Alpha < 0 {
		return fmt.Errorf("alpha (%.6f) must not be negative", c.Alpha)
	}

	if c.Beta < 0 || c.Beta > 1 {
		return fmt.Errorf("beta (%.6f) must be between 0 and 1", c.Beta)
	}

	if c.WindowSize <= 0 {
		return fmt.Errorf("window size (%d) must be positive", c.WindowSize)
	}

	if c.RandomnessFactor < 0 || c.RandomnessFactor > 1.0 {
		return fmt.Errorf("randomness factor (%.3f) must be between 0.0 and 1.0", c.RandomnessFactor)
	}

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
	fmt.Println("  aimd-simulator [flags]                   # Run with synthetic scenarios")
	fmt.Println("  aimd-simulator -scenario=full -graph     # Run specific scenario with charts")
	fmt.Println("  aimd-simulator -help                     # Show this help")
	fmt.Println()

	fmt.Println("Base Blockchain Integration:")
	fmt.Println("  aimd-simulator fetch-base <start> <end> <file>  # Fetch blockchain data")
	fmt.Println("    - Example: aimd-simulator fetch-base 12000000 12000100 data.json")
	fmt.Println("    - Downloads real Base blockchain data for analysis")
	fmt.Println("    - Supports concurrent fetching with retry logic")
	fmt.Println("    - Warns for large ranges (>10,000 blocks)")
	fmt.Println()
	fmt.Println("  aimd-simulator simulate-base <file> [flags]     # Simulate against real data")
	fmt.Println("    - Example: aimd-simulator simulate-base data.json -graph -gamma=0.1")
	fmt.Println("    - Runs AIMD algorithm against fetched blockchain data")
	fmt.Println("    - Supports all AIMD parameter flags")
	fmt.Println("    - Generates comparison charts with -graph")
	fmt.Println()

	fmt.Println("CORE AIMD PARAMETERS:")
	fmt.Println()

	fmt.Println("Block Configuration:")
	fmt.Println("  -target-block-size=15000000  Target block size in gas units")
	fmt.Printf("                               Default: %d (%.1f M gas)\n", p.config.TargetBlockSize, float64(p.config.TargetBlockSize)/1e6)
	fmt.Println("  -burst-multiplier=2.0        Max burst capacity as multiple of target")
	fmt.Printf("                               Default: %.1f (%.1fx target capacity)\n", p.config.BurstMultiplier, p.config.BurstMultiplier)
	fmt.Println("  -window-size=10              Number of blocks in analysis window")
	fmt.Printf("                               Default: %d blocks\n", p.config.WindowSize)
	fmt.Println()

	fmt.Println("Learning Rate Control:")
	fmt.Println("  -gamma=0.2                   Threshold for learning rate adjustment")
	fmt.Printf("                               Default: %.3f (deviation from target utilization)\n", p.config.Gamma)
	fmt.Println("  -max-learning-rate=0.5       Maximum learning rate")
	fmt.Printf("                               Default: %.3f (%.1f%% maximum adjustment)\n", p.config.MaxLearningRate, p.config.MaxLearningRate*100)
	fmt.Println("  -min-learning-rate=0.001     Minimum learning rate")
	fmt.Printf("                               Default: %.6f (%.3f%% minimum adjustment)\n", p.config.MinLearningRate, p.config.MinLearningRate*100)
	fmt.Println("  -alpha=0.01                  Additive increase factor")
	fmt.Printf("                               Default: %.6f (learning rate increase)\n", p.config.Alpha)
	fmt.Println("  -beta=0.9                    Multiplicative decrease factor")
	fmt.Printf("                               Default: %.3f (learning rate decay)\n", p.config.Beta)
	fmt.Println()

	fmt.Println("Fee Market Configuration:")
	fmt.Println("  -initial-base-fee=1000000000 Initial base fee in wei")
	fmt.Printf("                               Default: %d wei (%.3f Gwei)\n", p.config.InitialBaseFee, float64(p.config.InitialBaseFee)/1e9)
	fmt.Println("  -min-base-fee=0              Minimum base fee in wei")
	fmt.Printf("                               Default: %d wei (%.3f Gwei)\n", p.config.MinBaseFee, float64(p.config.MinBaseFee)/1e9)
	fmt.Println("  -initial-learning-rate=0.1   Initial learning rate")
	fmt.Printf("                               Default: %.3f (%.1f%% initial adjustment)\n", p.config.InitialLearningRate, p.config.InitialLearningRate*100)
	fmt.Println("  -delta=0.000001              Net gas delta coefficient")
	fmt.Printf("                               Default: %.9f (window gas delta impact)\n", p.config.Delta)
	fmt.Println()

	fmt.Println("Simulation Control:")
	fmt.Println("  -randomness=0.1              Gaussian noise level (0.0-1.0)")
	fmt.Printf("                               Default: %.3f (%.1f%% variation)\n", p.config.RandomnessFactor, p.config.RandomnessFactor*100)
	fmt.Println("  -scenario=all                Scenario to run")
	fmt.Printf("                               Default: %s\n", p.simConfig.Scenario)
	fmt.Println("                               Options: full, empty, stable, mixed, all")
	fmt.Println("                               - full:   Sustained high congestion (35 blocks)")
	fmt.Println("                               - empty:  Sustained low demand (35 blocks)")
	fmt.Println("                               - stable: Long-term stability (40 blocks)")
	fmt.Println("                               - mixed:  Realistic traffic patterns (80 blocks)")
	fmt.Println("                               - all:    Run all scenarios sequentially")
	fmt.Println("  -graph                       Generate visualization charts (PNG files)")
	fmt.Println("                               Creates fee evolution and comparison charts")
	fmt.Println()

	fmt.Println("EXAMPLE WORKFLOWS:")
	fmt.Println()

	fmt.Println("Quick Start:")
	fmt.Println("  aimd-simulator                           # Run with default settings")
	fmt.Println("  aimd-simulator -scenario=mixed -graph    # Test mixed traffic with charts")
	fmt.Println("  aimd-simulator -help                     # Show this help")
	fmt.Println()

	fmt.Println("Parameter Testing:")
	fmt.Println("  # Test burst capacity effects")
	fmt.Println("  aimd-simulator -burst-multiplier=1.5 -scenario=full")
	fmt.Println("  aimd-simulator -burst-multiplier=3.0 -scenario=full")
	fmt.Println()
	fmt.Println("  # Compare learning strategies")
	fmt.Println("  aimd-simulator -gamma=0.1 -alpha=0.02    # Aggressive response")
	fmt.Println("  aimd-simulator -gamma=0.5 -alpha=0.005   # Conservative response")
	fmt.Println()
	fmt.Println("  # Test randomness impact")
	fmt.Println("  aimd-simulator -randomness=0.0           # Deterministic")
	fmt.Println("  aimd-simulator -randomness=0.3           # High variation")
	fmt.Println()

	fmt.Println("Real Data Analysis:")
	fmt.Println("  # 1. Fetch blockchain data")
	fmt.Println("  aimd-simulator fetch-base 12000000 12001000 analysis.json")
	fmt.Println()
	fmt.Println("  # 2. Test different configurations")
	fmt.Println("  aimd-simulator simulate-base analysis.json -graph")
	fmt.Println("  aimd-simulator simulate-base analysis.json -gamma=0.1 -graph")
	fmt.Println("  aimd-simulator simulate-base analysis.json -burst-multiplier=2.5 -graph")
	fmt.Println()
	fmt.Println("  # 3. Performance comparison")
	fmt.Println("  aimd-simulator simulate-base analysis.json -window-size=5    # Fast")
	fmt.Println("  aimd-simulator simulate-base analysis.json -window-size=20   # Stable")
	fmt.Println()

	fmt.Println("Advanced Usage:")
	fmt.Println("  # Custom fee market setup")
	fmt.Println("  aimd-simulator -initial-base-fee=2000000000 -min-base-fee=500000000")
	fmt.Println()
	fmt.Println("  # High-precision delta tuning")
	fmt.Println("  aimd-simulator -delta=0.000005 -window-size=15")
	fmt.Println()
	fmt.Println("  # Large block size testing")
	fmt.Println("  aimd-simulator -target-block-size=30000000 -burst-multiplier=2.5")
	fmt.Println()

	fmt.Println("OUTPUT FILES:")
	fmt.Println("  When -graph is enabled, the following files are generated:")
	fmt.Println("  - chart_<scenario>.png           AIMD fee evolution charts")
	fmt.Println("  - base_comparison_<range>.png    AIMD vs Base fee comparison")
	fmt.Println("  - base_comparison_<range>_gas.png Gas usage analysis")
	fmt.Println()

	fmt.Println("PERFORMANCE NOTES:")
	fmt.Println("  - Basic simulations run in seconds")
	fmt.Println("  - Data fetching: ~2-5 sec for 10-100 blocks, minutes for 1000+ blocks")
	fmt.Println("  - Large ranges (>10,000 blocks) require confirmation")
	fmt.Println("  - Concurrent fetching with exponential backoff retry")
	fmt.Println("  - No-gaps guarantee ensures complete datasets")
	fmt.Println()

	fmt.Println("For detailed algorithm description and parameter explanations,")
	fmt.Println("see the project README.md file.")
}
