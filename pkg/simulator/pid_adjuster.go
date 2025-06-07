package simulator

import (
	"fmt"
	"math"
)

// PIDConfig holds configuration specific to PID controller
type PIDConfig struct {
	TargetBlockSize uint64
	BurstMultiplier float64
	InitialBaseFee  uint64
	MinBaseFee      uint64

	// PID parameters
	Kp float64 // Proportional gain
	Ki float64 // Integral gain
	Kd float64 // Derivative gain

	// Integral windup prevention
	MaxIntegral float64
	MinIntegral float64

	// Output limits
	MaxFeeChange float64 // Maximum fee change per block (as ratio)
	WindowSize   int     // Window for derivative calculation
}

// DefaultPIDConfig returns the default PID configuration
func DefaultPIDConfig() *PIDConfig {
	return &PIDConfig{
		TargetBlockSize: 15_000_000,
		BurstMultiplier: 2.0,
		InitialBaseFee:  1_000_000_000,
		MinBaseFee:      0,

		// Conservative PID settings
		Kp: 0.1,  // Proportional gain
		Ki: 0.01, // Integral gain
		Kd: 0.05, // Derivative gain

		MaxIntegral:  1000.0,
		MinIntegral:  -1000.0,
		MaxFeeChange: 0.25, // 25% max change
		WindowSize:   3,    // Look back 3 blocks for derivative
	}
}

// Implement AdjusterConfig interface
func (c *PIDConfig) GetTargetBlockSize() uint64  { return c.TargetBlockSize }
func (c *PIDConfig) GetBurstMultiplier() float64 { return c.BurstMultiplier }
func (c *PIDConfig) GetInitialBaseFee() uint64   { return c.InitialBaseFee }
func (c *PIDConfig) GetMinBaseFee() uint64       { return c.MinBaseFee }

// PIDFeeAdjuster implements a PID controller for fee adjustment
type PIDFeeAdjuster struct {
	config  *PIDConfig
	blocks  []Block
	baseFee uint64

	// PID state
	integral     float64   // Integral term accumulator
	lastError    float64   // Previous error for derivative calculation
	errorHistory []float64 // Error history for derivative calculation
}

// NewPIDFeeAdjuster creates a new PID fee adjuster
func NewPIDFeeAdjuster(cfg *PIDConfig) FeeAdjuster {
	return &PIDFeeAdjuster{
		config:       cfg,
		blocks:       make([]Block, 0),
		baseFee:      cfg.InitialBaseFee,
		integral:     0.0,
		lastError:    0.0,
		errorHistory: make([]float64, 0),
	}
}

// GetMaxBlockSize returns the current maximum block size
func (fa *PIDFeeAdjuster) GetMaxBlockSize() uint64 {
	return CalculateMaxBlockSize(fa.config.TargetBlockSize, fa.config.BurstMultiplier)
}

// ProcessBlock processes a new block using PID control
func (fa *PIDFeeAdjuster) ProcessBlock(gasUsed uint64) {
	// Add the new block
	block := Block{
		Number:  len(fa.blocks) + 1,
		GasUsed: gasUsed,
		BaseFee: fa.baseFee,
	}
	fa.blocks = append(fa.blocks, block)

	// Calculate error (utilization deviation from target)
	targetUtilization := 1.0
	currentUtilization := float64(gasUsed) / float64(fa.config.TargetBlockSize)
	delta := currentUtilization - targetUtilization

	// Update PID components
	fa.updatePIDState(delta)

	// Calculate output and adjust base fee
	fa.adjustBaseFeePID(delta)
}

// updatePIDState updates the PID controller state
func (fa *PIDFeeAdjuster) updatePIDState(delta float64) {
	// Update integral term with windup protection
	fa.integral += delta
	fa.integral = ClampFloat64(fa.integral, fa.config.MinIntegral, fa.config.MaxIntegral)

	// Update error history for derivative calculation
	fa.errorHistory = append(fa.errorHistory, delta)
	if len(fa.errorHistory) > fa.config.WindowSize {
		fa.errorHistory = fa.errorHistory[1:]
	}

	fa.lastError = delta
}

// calculateDerivative calculates the derivative term
func (fa *PIDFeeAdjuster) calculateDerivative() float64 {
	if len(fa.errorHistory) < 2 {
		return 0.0
	}

	if len(fa.errorHistory) < fa.config.WindowSize {
		// Simple derivative with available data
		return fa.errorHistory[len(fa.errorHistory)-1] - fa.errorHistory[len(fa.errorHistory)-2]
	}

	// Linear regression slope over the window for smoother derivative
	n := float64(len(fa.errorHistory))
	var sumX, sumY, sumXY, sumX2 float64

	for i, err := range fa.errorHistory {
		x := float64(i)
		y := err
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope (derivative)
	denominator := n*sumX2 - sumX*sumX
	if math.Abs(denominator) < 1e-10 {
		return 0.0
	}

	return (n*sumXY - sumX*sumY) / denominator
}

// adjustBaseFeePID adjusts the base fee using PID control
func (fa *PIDFeeAdjuster) adjustBaseFeePID(error float64) {
	// Calculate PID terms
	proportional := fa.config.Kp * error
	integral := fa.config.Ki * fa.integral
	derivative := fa.config.Kd * fa.calculateDerivative()

	// Calculate total control output
	controlOutput := proportional + integral + derivative

	// Limit the control output to prevent excessive changes
	maxChange := fa.config.MaxFeeChange
	controlOutput = ClampFloat64(controlOutput, -maxChange, maxChange)

	// Apply the control output to the base fee
	newBaseFee := float64(fa.baseFee) * (1.0 + controlOutput)

	// Ensure base fee doesn't go below minimum
	if newBaseFee < float64(fa.config.MinBaseFee) {
		newBaseFee = float64(fa.config.MinBaseFee)
	}

	fa.baseFee = uint64(newBaseFee)
}

// GetCurrentState returns the current state of the fee adjuster
func (fa *PIDFeeAdjuster) GetCurrentState() State {
	var targetUtilization float64
	var burstUtilization float64

	if len(fa.blocks) > 0 {
		// Calculate utilization based on recent blocks
		windowSize := fa.config.WindowSize
		if windowSize > len(fa.blocks) {
			windowSize = len(fa.blocks)
		}

		targetUtilization = CalculateTargetUtilization(fa.blocks, windowSize, fa.config.TargetBlockSize)
		burstUtilization = CalculateBurstUtilization(fa.blocks, windowSize, fa.GetMaxBlockSize())
	}

	// Calculate effective learning rate based on last 2 blocks
	var effectiveLearningRate float64
	if len(fa.blocks) >= 2 {
		lastBlock := fa.blocks[len(fa.blocks)-1]
		prevBlock := fa.blocks[len(fa.blocks)-2]

		// Calculate rate of change in base fee
		var baseFeeChange float64
		if lastBlock.BaseFee > prevBlock.BaseFee {
			baseFeeChange = float64(lastBlock.BaseFee-prevBlock.BaseFee) / float64(prevBlock.BaseFee)
		} else {
			baseFeeChange = float64(prevBlock.BaseFee-lastBlock.BaseFee) / float64(prevBlock.BaseFee)
		}
		fmt.Printf("Base fee change: %f\n", baseFeeChange)

		// Calculate excess utilization
		excessUtilization := (float64(lastBlock.GasUsed) - float64(fa.config.TargetBlockSize)) / float64(fa.config.TargetBlockSize)
		fmt.Printf("Excess utilization: %f\n", excessUtilization)

		// Effective learning rate is the ratio of base fee change to utilization change
		if math.Abs(excessUtilization) > 1e-10 {
			effectiveLearningRate = math.Abs(baseFeeChange / excessUtilization)
			fmt.Printf("Effective learning rate: %f\n", effectiveLearningRate)
		}
	}

	return State{
		BaseFee:           fa.baseFee,
		LearningRate:      effectiveLearningRate,
		TargetUtilization: targetUtilization,
		BurstUtilization:  burstUtilization,
	}
}

// GetBlocks returns a copy of the blocks processed so far
func (fa *PIDFeeAdjuster) GetBlocks() []Block {
	blocks := make([]Block, len(fa.blocks))
	copy(blocks, fa.blocks)
	return blocks
}

// Reset resets the fee adjuster to its initial state
func (fa *PIDFeeAdjuster) Reset() {
	fa.blocks = fa.blocks[:0]
	fa.baseFee = fa.config.InitialBaseFee
	fa.integral = 0.0
	fa.lastError = 0.0
	fa.errorHistory = fa.errorHistory[:0]
}
