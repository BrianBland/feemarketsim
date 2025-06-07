package simulator

// import (
// 	"math/rand"
// 	"time"
// )

// // RNGConfig holds configuration for the RNG wrapper
// type RNGConfig struct {
// 	// Base randomness factors
// 	GasRandomnessFactor float64 // Additional randomness for gas usage

// 	// Burst mode
// 	BurstMode {
// 		Probability float64 // Probability of entering burst mode each block (disabled if 0)
// 		DurationMin int     // Minimum burst duration (blocks)
// 		DurationMax int     // Maximum burst duration (blocks)
// 		Intensity   float64 // Multiplier for gas usage during bursts
// 	}

// 	// Fee adjustment randomness
// 	EnableFeeJitter    bool    // Enable random fee adjustment jitter
// 	FeeJitterAmplitude float64 // Maximum jitter as fraction of fee change

// 	// Network conditions simulation
// 	EnableNetworkDelays bool    // Simulate network propagation delays
// 	DelayProbability    float64 // Probability of delay per block
// 	MaxDelayBlocks      int     // Maximum delay in blocks
// }

// // DefaultRNGConfig returns the default RNG configuration
// func DefaultRNGConfig() *RNGConfig {
// 	return &RNGConfig{
// 		GasRandomnessFactor: 0.05, // 5% additional gas randomness

// 		EnableBurstModes: true,
// 		BurstProbability: 0.1, // 10% chance per block
// 		BurstDurationMin: 2,
// 		BurstDurationMax: 5,
// 		BurstIntensity:   1.5, // 150% gas usage during bursts

// 		EnableFeeJitter:    true,
// 		FeeJitterAmplitude: 0.05, // 5% jitter

// 		EnableNetworkDelays: false,
// 		DelayProbability:    0.05, // 5% chance per block
// 		MaxDelayBlocks:      2,
// 	}
// }

// // RNGFeeAdjuster wraps another FeeAdjuster with additional randomness
// type RNGFeeAdjuster struct {
// 	wrapped FeeAdjuster
// 	config  *RNGConfig
// 	rng     *rand.Rand

// 	// Burst mode state
// 	inBurstMode     bool
// 	burstBlocksLeft int

// 	// Delay simulation state
// 	delayedBlocks   []uint64 // Queue of delayed gas usage values
// 	delayBlocksLeft int

// 	// State tracking
// 	processedBlocks int
// }

// // NewRNGFeeAdjuster creates a new RNG wrapper around another fee adjuster
// func NewRNGFeeAdjuster(wrapped FeeAdjuster, config *RNGConfig) FeeAdjuster {
// 	return &RNGFeeAdjuster{
// 		wrapped:         wrapped,
// 		config:          config,
// 		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
// 		inBurstMode:     false,
// 		burstBlocksLeft: 0,
// 		delayedBlocks:   make([]uint64, 0),
// 		delayBlocksLeft: 0,
// 		processedBlocks: 0,
// 	}
// }

// // ProcessBlock processes a block with additional randomness effects
// func (rfa *RNGFeeAdjuster) ProcessBlock(gasUsed uint64) {
// 	rfa.processedBlocks++

// 	// Apply gas usage randomness
// 	modifiedGasUsed := rfa.applyGasRandomness(gasUsed)

// 	// Handle burst mode
// 	modifiedGasUsed = rfa.applyBurstMode(modifiedGasUsed)

// 	// Handle network delays
// 	if rfa.config.EnableNetworkDelays {
// 		rfa.handleNetworkDelays(modifiedGasUsed)
// 		return
// 	}

// 	// Process the block normally
// 	rfa.processBlockWithFeeJitter(modifiedGasUsed)
// }

// // applyGasRandomness adds randomness to gas usage
// func (rfa *RNGFeeAdjuster) applyGasRandomness(gasUsed uint64) uint64 {
// 	if rfa.config.GasRandomnessFactor == 0 {
// 		return gasUsed
// 	}

// 	// Apply gaussian noise
// 	noise := rfa.rng.NormFloat64() * rfa.config.GasRandomnessFactor
// 	multiplier := 1.0 + noise

// 	result := uint64(float64(gasUsed) * multiplier)

// 	// Ensure we don't exceed max block size
// 	maxSize := rfa.wrapped.GetMaxBlockSize()
// 	if result > maxSize {
// 		result = maxSize
// 	}

// 	return result
// }

// // applyBurstMode handles burst mode logic
// func (rfa *RNGFeeAdjuster) applyBurstMode(gasUsed uint64) uint64 {
// 	if !rfa.config.EnableBurstModes {
// 		return gasUsed
// 	}

// 	// Update burst mode state
// 	if rfa.inBurstMode {
// 		rfa.burstBlocksLeft--
// 		if rfa.burstBlocksLeft <= 0 {
// 			rfa.inBurstMode = false
// 		}
// 	} else {
// 		// Check if we should enter burst mode
// 		if rfa.rng.Float64() < rfa.config.BurstProbability {
// 			rfa.inBurstMode = true
// 			duration := rfa.config.BurstDurationMin +
// 				rfa.rng.Intn(rfa.config.BurstDurationMax-rfa.config.BurstDurationMin+1)
// 			rfa.burstBlocksLeft = duration
// 		}
// 	}

// 	// Apply burst intensity
// 	if rfa.inBurstMode {
// 		result := uint64(float64(gasUsed) * rfa.config.BurstIntensity)
// 		maxSize := rfa.wrapped.GetMaxBlockSize()
// 		if result > maxSize {
// 			result = maxSize
// 		}
// 		return result
// 	}

// 	return gasUsed
// }

// // handleNetworkDelays simulates network propagation delays
// func (rfa *RNGFeeAdjuster) handleNetworkDelays(gasUsed uint64) {
// 	// Add current block to delay queue if delay occurs
// 	if rfa.rng.Float64() < rfa.config.DelayProbability {
// 		rfa.delayedBlocks = append(rfa.delayedBlocks, gasUsed)
// 		rfa.delayBlocksLeft = rfa.config.MaxDelayBlocks
// 		return
// 	}

// 	// Process delayed blocks first
// 	if len(rfa.delayedBlocks) > 0 && rfa.delayBlocksLeft <= 0 {
// 		delayedGas := rfa.delayedBlocks[0]
// 		rfa.delayedBlocks = rfa.delayedBlocks[1:]
// 		rfa.processBlockWithFeeJitter(delayedGas)
// 	}

// 	if rfa.delayBlocksLeft > 0 {
// 		rfa.delayBlocksLeft--
// 	}

// 	// Process current block normally
// 	rfa.processBlockWithFeeJitter(gasUsed)
// }

// // processBlockWithFeeJitter processes a block and applies fee jitter
// func (rfa *RNGFeeAdjuster) processBlockWithFeeJitter(gasUsed uint64) {
// 	// Get state before processing
// 	stateBefore := rfa.wrapped.GetCurrentState()

// 	// Process the block
// 	rfa.wrapped.ProcessBlock(gasUsed)

// 	// Apply fee jitter if enabled
// 	if rfa.config.EnableFeeJitter && rfa.config.FeeJitterAmplitude > 0 {
// 		rfa.applyFeeJitter(stateBefore)
// 	}
// }

// // applyFeeJitter adds randomness to fee adjustments
// func (rfa *RNGFeeAdjuster) applyFeeJitter(stateBefore State) {
// 	stateAfter := rfa.wrapped.GetCurrentState()

// 	// Calculate the fee change that occurred
// 	feeChange := int64(stateAfter.BaseFee) - int64(stateBefore.BaseFee)
// 	if feeChange == 0 {
// 		return
// 	}

// 	// Apply jitter to the fee change
// 	jitter := rfa.rng.NormFloat64() * rfa.config.FeeJitterAmplitude
// 	jitteredChange := float64(feeChange) * (1.0 + jitter)

// 	// Calculate new base fee with jitter
// 	newBaseFee := int64(stateBefore.BaseFee) + int64(jitteredChange)

// 	// Ensure fee doesn't go below minimum (we need to access the wrapped adjuster's config)
// 	// This is a simplification - in a real implementation, we'd need better access to min fee
// 	if newBaseFee < 0 {
// 		newBaseFee = 0
// 	}

// 	// Force the new fee (this is a hack - in reality we'd need a better way to override the fee)
// 	// For now, we'll just let the normal fee adjustment stand
// }

// // GetCurrentState returns the current state from the wrapped adjuster
// func (rfa *RNGFeeAdjuster) GetCurrentState() State {
// 	state := rfa.wrapped.GetCurrentState()

// 	// Add RNG-specific information to learning rate
// 	if rfa.inBurstMode {
// 		state.LearningRate *= 1.1 // Indicate burst mode is active
// 	}

// 	return state
// }

// // GetMaxBlockSize delegates to the wrapped adjuster
// func (rfa *RNGFeeAdjuster) GetMaxBlockSize() uint64 {
// 	return rfa.wrapped.GetMaxBlockSize()
// }

// // GetBlocks delegates to the wrapped adjuster
// func (rfa *RNGFeeAdjuster) GetBlocks() []Block {
// 	return rfa.wrapped.GetBlocks()
// }

// // Reset resets both the wrapper and wrapped adjuster
// func (rfa *RNGFeeAdjuster) Reset() {
// 	rfa.wrapped.Reset()
// 	rfa.inBurstMode = false
// 	rfa.burstBlocksLeft = 0
// 	rfa.delayedBlocks = rfa.delayedBlocks[:0]
// 	rfa.delayBlocksLeft = 0
// 	rfa.processedBlocks = 0
// }
