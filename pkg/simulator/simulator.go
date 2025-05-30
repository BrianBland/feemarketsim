package simulator

import (
	"math"
	"math/rand"
	"time"

	"github.com/brianbland/feemarketsim/pkg/config"
)

// Block represents a block with its gas usage
type Block struct {
	Number  int
	GasUsed uint64
	BaseFee uint64
}

// State represents the current state of the fee adjuster
type State struct {
	BaseFee           uint64
	LearningRate      float64
	TargetUtilization float64
	BurstUtilization  float64
}

// FeeAdjuster implements the AIMD fee adjustment mechanism
type FeeAdjuster struct {
	config       config.Config
	blocks       []Block
	learningRate float64
	baseFee      uint64
	rng          *rand.Rand
}

// NewFeeAdjuster creates a new fee adjuster with the given configuration
func NewFeeAdjuster(cfg config.Config) *FeeAdjuster {
	return &FeeAdjuster{
		config:       cfg,
		blocks:       make([]Block, 0),
		learningRate: cfg.InitialLearningRate,
		baseFee:      cfg.InitialBaseFee,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetMaxBlockSize returns the current maximum block size (target * burst multiplier)
func (fa *FeeAdjuster) GetMaxBlockSize() uint64 {
	return uint64(float64(fa.config.TargetBlockSize) * fa.config.BurstMultiplier)
}

// sumBlockSizesInWindow returns the sum of the block sizes in the window
func (fa *FeeAdjuster) sumBlockSizesInWindow() uint64 {
	windowStart := len(fa.blocks) - fa.config.WindowSize
	if windowStart < 0 {
		windowStart = 0
	}

	var sum uint64
	for i := windowStart; i < len(fa.blocks); i++ {
		sum += fa.blocks[i].GasUsed
	}
	return sum
}

// netGasDelta returns the net gas difference between every block in the window and the target block size
func (fa *FeeAdjuster) netGasDelta() int64 {
	windowStart := len(fa.blocks) - fa.config.WindowSize
	if windowStart < 0 {
		windowStart = 0
	}

	var netDelta int64
	for i := windowStart; i < len(fa.blocks); i++ {
		netDelta += int64(fa.blocks[i].GasUsed) - int64(fa.config.TargetBlockSize)
	}
	return netDelta
}

// ProcessBlock processes a new block and updates the base fee and learning rate
func (fa *FeeAdjuster) ProcessBlock(gasUsed uint64) {
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
func (fa *FeeAdjuster) adjustLearningRate() {
	// Calculate target utilization (relative to target, not max)
	targetUtilization := float64(fa.sumBlockSizesInWindow()) /
		(float64(fa.config.WindowSize) * float64(fa.config.TargetBlockSize))

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
func (fa *FeeAdjuster) adjustBaseFee(gasUsed uint64) {
	currentBlockSize := float64(gasUsed)
	targetBlockSize := float64(fa.config.TargetBlockSize)

	adjustment := fa.learningRate * (currentBlockSize - targetBlockSize) / targetBlockSize
	deltaAdjustment := fa.config.Delta * float64(fa.netGasDelta())

	newBaseFee := float64(fa.baseFee)*(1+adjustment) + deltaAdjustment

	// Ensure base fee doesn't go negative
	if newBaseFee < float64(fa.config.MinBaseFee) {
		newBaseFee = float64(fa.config.MinBaseFee)
	}

	fa.baseFee = uint64(newBaseFee)
}

// GetCurrentState returns the current state of the fee adjuster
func (fa *FeeAdjuster) GetCurrentState() State {
	var targetUtilization float64
	var burstUtilization float64

	if len(fa.blocks) >= fa.config.WindowSize {
		windowSum := float64(fa.sumBlockSizesInWindow())
		windowSize := float64(fa.config.WindowSize)
		targetUtilization = windowSum / (windowSize * float64(fa.config.TargetBlockSize))
		burstUtilization = windowSum / (windowSize * float64(fa.GetMaxBlockSize()))
	}

	return State{
		BaseFee:           fa.baseFee,
		LearningRate:      fa.learningRate,
		TargetUtilization: targetUtilization,
		BurstUtilization:  burstUtilization,
	}
}

// AddRandomness adds gaussian noise to a gas usage value
func (fa *FeeAdjuster) AddRandomness(gasUsed uint64) uint64 {
	if fa.config.RandomnessFactor == 0 {
		return gasUsed
	}

	// Generate gaussian noise with mean=0, std=randomnessFactor
	noise := fa.rng.NormFloat64() * fa.config.RandomnessFactor
	multiplier := 1.0 + noise

	// Ensure we don't go below 0 or above burst capacity
	result := uint64(float64(gasUsed) * multiplier)
	maxAllowed := fa.GetMaxBlockSize()

	if result > maxAllowed {
		result = maxAllowed
	}

	return result
}

// GetBlocks returns a copy of the blocks processed so far
func (fa *FeeAdjuster) GetBlocks() []Block {
	blocks := make([]Block, len(fa.blocks))
	copy(blocks, fa.blocks)
	return blocks
}

// Reset resets the fee adjuster to its initial state
func (fa *FeeAdjuster) Reset() {
	fa.blocks = fa.blocks[:0]
	fa.learningRate = fa.config.InitialLearningRate
	fa.baseFee = fa.config.InitialBaseFee
}

// SetSeed sets the random seed for reproducible testing
func (fa *FeeAdjuster) SetSeed(seed int64) {
	fa.rng = rand.New(rand.NewSource(seed))
}
