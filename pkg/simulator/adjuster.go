package simulator

// Block represents a block with its gas usage and fee information
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

// FeeAdjuster is the interface that all fee adjustment algorithms must implement
type FeeAdjuster interface {
	// ProcessBlock processes a new block and updates the internal state
	ProcessBlock(gasUsed uint64)

	// GetCurrentState returns the current state of the fee adjuster
	GetCurrentState() State

	// GetMaxBlockSize returns the current maximum block size
	GetMaxBlockSize() uint64

	// GetBlocks returns a copy of the blocks processed so far
	GetBlocks() []Block

	// Reset resets the fee adjuster to its initial state
	Reset()
}

// AdjusterConfig represents the base configuration for all adjusters
type AdjusterConfig interface {
	GetTargetBlockSize() uint64
	GetBurstMultiplier() float64
	GetInitialBaseFee() uint64
	GetMinBaseFee() uint64
}

// CalculateMaxBlockSize returns the maximum block size based on target and burst multiplier
func CalculateMaxBlockSize(targetBlockSize uint64, burstMultiplier float64) uint64 {
	return uint64(float64(targetBlockSize) * burstMultiplier)
}

// SumBlockSizesInWindow returns the sum of block sizes in the given window
func SumBlockSizesInWindow(blocks []Block, windowSize int) uint64 {
	windowStart := len(blocks) - windowSize
	if windowStart < 0 {
		windowStart = 0
	}

	var sum uint64
	for i := windowStart; i < len(blocks); i++ {
		sum += blocks[i].GasUsed
	}
	return sum
}

// NetGasDelta returns the net gas difference between every block in the window and the target block size
func NetGasDelta(blocks []Block, windowSize int, targetBlockSize uint64) int64 {
	windowStart := len(blocks) - windowSize
	if windowStart < 0 {
		windowStart = 0
	}

	var netDelta int64
	for i := windowStart; i < len(blocks); i++ {
		netDelta += int64(blocks[i].GasUsed) - int64(targetBlockSize)
	}
	return netDelta
}

// CalculateTargetUtilization calculates utilization relative to target capacity
func CalculateTargetUtilization(blocks []Block, windowSize int, targetBlockSize uint64) float64 {
	if len(blocks) < windowSize {
		return 0
	}

	windowSum := float64(SumBlockSizesInWindow(blocks, windowSize))
	windowSizeFloat := float64(windowSize)
	return windowSum / (windowSizeFloat * float64(targetBlockSize))
}

// CalculateBurstUtilization calculates utilization relative to maximum burst capacity
func CalculateBurstUtilization(blocks []Block, windowSize int, maxBlockSize uint64) float64 {
	if len(blocks) < windowSize {
		return 0
	}

	windowSum := float64(SumBlockSizesInWindow(blocks, windowSize))
	windowSizeFloat := float64(windowSize)
	return windowSum / (windowSizeFloat * float64(maxBlockSize))
}

// ClampUint64 ensures value is within the specified bounds
func ClampUint64(value, min, max uint64) uint64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// ClampFloat64 ensures value is within the specified bounds
func ClampFloat64(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
