package simulator

// EIP1559Config holds configuration specific to EIP-1559
type EIP1559Config struct {
	TargetBlockSize uint64
	BurstMultiplier float64
	InitialBaseFee  uint64
	MinBaseFee      uint64
	MaxFeeChange    float64 // Maximum fee change per block (1/8 = 0.125)
}

// DefaultEIP1559Config returns the default EIP-1559 configuration
func DefaultEIP1559Config() *EIP1559Config {
	return &EIP1559Config{
		TargetBlockSize: 15_000_000,
		BurstMultiplier: 2.0,
		InitialBaseFee:  1_000_000_000,
		MinBaseFee:      0,
		MaxFeeChange:    0.125, // 1/8 as per EIP-1559
	}
}

// Implement AdjusterConfig interface
func (c *EIP1559Config) GetTargetBlockSize() uint64  { return c.TargetBlockSize }
func (c *EIP1559Config) GetBurstMultiplier() float64 { return c.BurstMultiplier }
func (c *EIP1559Config) GetInitialBaseFee() uint64   { return c.InitialBaseFee }
func (c *EIP1559Config) GetMinBaseFee() uint64       { return c.MinBaseFee }

// EIP1559FeeAdjuster implements the standard EIP-1559 fee adjustment mechanism
type EIP1559FeeAdjuster struct {
	config  *EIP1559Config
	blocks  []Block
	baseFee uint64
}

// NewEIP1559FeeAdjuster creates a new EIP-1559 fee adjuster
func NewEIP1559FeeAdjuster(cfg *EIP1559Config) FeeAdjuster {
	return &EIP1559FeeAdjuster{
		config:  cfg,
		blocks:  make([]Block, 0),
		baseFee: cfg.InitialBaseFee,
	}
}

// GetMaxBlockSize returns the current maximum block size
func (fa *EIP1559FeeAdjuster) GetMaxBlockSize() uint64 {
	return CalculateMaxBlockSize(fa.config.TargetBlockSize, fa.config.BurstMultiplier)
}

// ProcessBlock processes a new block according to EIP-1559 rules
func (fa *EIP1559FeeAdjuster) ProcessBlock(gasUsed uint64) {
	// Add the new block
	block := Block{
		Number:  len(fa.blocks) + 1,
		GasUsed: gasUsed,
		BaseFee: fa.baseFee,
	}
	fa.blocks = append(fa.blocks, block)

	// EIP-1559 adjusts based on the current block only
	fa.adjustBaseFeeEIP1559(gasUsed)
}

// adjustBaseFeeEIP1559 adjusts the base fee according to EIP-1559 formula
func (fa *EIP1559FeeAdjuster) adjustBaseFeeEIP1559(gasUsed uint64) {
	targetGas := fa.config.TargetBlockSize

	if gasUsed == targetGas {
		// No change needed
		return
	}

	// Calculate the fee change
	gasUsedDelta := int64(gasUsed) - int64(targetGas)
	baseFeeChange := int64(fa.baseFee) * gasUsedDelta / int64(targetGas) / 8

	// Apply the change
	newBaseFee := int64(fa.baseFee) + baseFeeChange

	// Ensure base fee doesn't go below minimum
	if newBaseFee < int64(fa.config.MinBaseFee) {
		newBaseFee = int64(fa.config.MinBaseFee)
	}

	fa.baseFee = uint64(newBaseFee)
}

// GetCurrentState returns the current state of the fee adjuster
func (fa *EIP1559FeeAdjuster) GetCurrentState() State {
	var targetUtilization float64
	var burstUtilization float64

	if len(fa.blocks) > 0 {
		// EIP-1559 only considers the last block
		lastBlock := fa.blocks[len(fa.blocks)-1]
		targetUtilization = float64(lastBlock.GasUsed) / float64(fa.config.TargetBlockSize)
		burstUtilization = float64(lastBlock.GasUsed) / float64(fa.GetMaxBlockSize())
	}

	return State{
		BaseFee:           fa.baseFee,
		LearningRate:      fa.config.MaxFeeChange, // Fixed learning rate for EIP-1559
		TargetUtilization: targetUtilization,
		BurstUtilization:  burstUtilization,
	}
}

// GetBlocks returns a copy of the blocks processed so far
func (fa *EIP1559FeeAdjuster) GetBlocks() []Block {
	blocks := make([]Block, len(fa.blocks))
	copy(blocks, fa.blocks)
	return blocks
}

// Reset resets the fee adjuster to its initial state
func (fa *EIP1559FeeAdjuster) Reset() {
	fa.blocks = fa.blocks[:0]
	fa.baseFee = fa.config.InitialBaseFee
}
