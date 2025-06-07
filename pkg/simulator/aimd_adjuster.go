package simulator

import (
	"math"
)

// AIMDConfig represents the configuration for the AIMD fee adjuster
type AIMDConfig struct {
	TargetBlockSize     uint64
	BurstMultiplier     float64
	InitialBaseFee      uint64
	MinBaseFee          uint64
	WindowSize          int
	Gamma               float64
	InitialLearningRate float64
	MaxLearningRate     float64
	MinLearningRate     float64
	Alpha               float64
	Beta                float64
	Delta               float64
}

// DefaultAIMDConfig returns the default configuration for the AIMD fee adjuster
func DefaultAIMDConfig() *AIMDConfig {
	return &AIMDConfig{
		TargetBlockSize:     15_000_000,
		BurstMultiplier:     2.0,
		InitialBaseFee:      1_000_000_000,
		MinBaseFee:          0,
		WindowSize:          10,
		Gamma:               0.25,
		InitialLearningRate: 0.1,
		MaxLearningRate:     0.5,
		MinLearningRate:     0.001,
		Alpha:               0.01,
		Beta:                0.9,
		Delta:               0,
	}
}

// Implement AdjusterConfig interface
func (c *AIMDConfig) GetTargetBlockSize() uint64  { return c.TargetBlockSize }
func (c *AIMDConfig) GetBurstMultiplier() float64 { return c.BurstMultiplier }
func (c *AIMDConfig) GetInitialBaseFee() uint64   { return c.InitialBaseFee }
func (c *AIMDConfig) GetMinBaseFee() uint64       { return c.MinBaseFee }

// AIMDFeeAdjuster implements the AIMD fee adjustment mechanism
type AIMDFeeAdjuster struct {
	config       *AIMDConfig
	blocks       []Block
	learningRate float64
	baseFee      uint64
}

// NewAIMDFeeAdjuster creates a new AIMD fee adjuster with the given configuration
func NewAIMDFeeAdjuster(cfg *AIMDConfig) FeeAdjuster {
	return &AIMDFeeAdjuster{
		config:       cfg,
		blocks:       make([]Block, 0),
		learningRate: cfg.InitialLearningRate,
		baseFee:      cfg.InitialBaseFee,
	}
}

// GetMaxBlockSize returns the current maximum block size (target * burst multiplier)
func (fa *AIMDFeeAdjuster) GetMaxBlockSize() uint64 {
	return CalculateMaxBlockSize(fa.config.TargetBlockSize, fa.config.BurstMultiplier)
}

// ProcessBlock processes a new block and updates the base fee and learning rate
func (fa *AIMDFeeAdjuster) ProcessBlock(gasUsed uint64) {
	// Add the new block
	block := Block{
		Number:  len(fa.blocks) + 1,
		GasUsed: gasUsed,
		BaseFee: fa.baseFee,
	}
	fa.blocks = append(fa.blocks, block)

	// Only adjust if we have enough blocks for a full window
	if len(fa.blocks) < fa.config.WindowSize {
		return
	}

	fa.adjustLearningRate()
	fa.adjustBaseFee(gasUsed)
}

// adjustLearningRate adjusts the learning rate based on target utilization deviation
func (fa *AIMDFeeAdjuster) adjustLearningRate() {
	// Calculate target utilization (relative to target, not max)
	targetUtilization := CalculateTargetUtilization(fa.blocks, fa.config.WindowSize, fa.config.TargetBlockSize)

	// Adjust learning rate based on target utilization deviation
	utilizationDeviation := math.Abs(targetUtilization - 1.0)

	if utilizationDeviation > fa.config.Gamma {
		// Additive increase when far from target
		fa.learningRate = math.Min(fa.config.MaxLearningRate,
			fa.config.Alpha+fa.learningRate)
	} else {
		// Multiplicative decrease when near target
		fa.learningRate = math.Max(fa.config.MinLearningRate,
			fa.config.Beta*fa.learningRate)
	}
}

// adjustBaseFee calculates and updates the base fee
func (fa *AIMDFeeAdjuster) adjustBaseFee(gasUsed uint64) {
	currentBlockSize := float64(gasUsed)
	targetBlockSize := float64(fa.config.TargetBlockSize)

	adjustment := fa.learningRate * (currentBlockSize - targetBlockSize) / targetBlockSize
	deltaAdjustment := fa.config.Delta * float64(NetGasDelta(fa.blocks, fa.config.WindowSize, fa.config.TargetBlockSize))

	newBaseFee := float64(fa.baseFee)*(1+adjustment) + deltaAdjustment

	// Ensure base fee doesn't go negative
	if newBaseFee < float64(fa.config.MinBaseFee) {
		newBaseFee = float64(fa.config.MinBaseFee)
	}

	fa.baseFee = uint64(newBaseFee)
}

// GetCurrentState returns the current state of the fee adjuster
func (fa *AIMDFeeAdjuster) GetCurrentState() State {
	var targetUtilization float64
	var burstUtilization float64

	if len(fa.blocks) >= fa.config.WindowSize {
		targetUtilization = CalculateTargetUtilization(fa.blocks, fa.config.WindowSize, fa.config.TargetBlockSize)
		burstUtilization = CalculateBurstUtilization(fa.blocks, fa.config.WindowSize, fa.GetMaxBlockSize())
	}

	return State{
		BaseFee:           fa.baseFee,
		LearningRate:      fa.learningRate,
		TargetUtilization: targetUtilization,
		BurstUtilization:  burstUtilization,
	}
}

// GetBlocks returns a copy of the blocks processed so far
func (fa *AIMDFeeAdjuster) GetBlocks() []Block {
	blocks := make([]Block, len(fa.blocks))
	copy(blocks, fa.blocks)
	return blocks
}

// Reset resets the fee adjuster to its initial state
func (fa *AIMDFeeAdjuster) Reset() {
	fa.blocks = fa.blocks[:0]
	fa.learningRate = fa.config.InitialLearningRate
	fa.baseFee = fa.config.InitialBaseFee
}
