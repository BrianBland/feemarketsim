package simulator

import (
	"fmt"
	"math"
	"time"
)

// DAMetrics represents L1 Data Availability metrics for a time window
type DAMetrics struct {
	Timestamp       time.Time
	L1GasPrice      uint64  // L1 gas price in wei
	BlobPrice       uint64  // Blob gas price in wei
	DAUsage         uint64  // DA bytes used in this window
	DACapacity      uint64  // Max DA bytes available
	BatchCost       uint64  // Cost to submit batch in wei
	BatchEfficiency float64 // Utilization efficiency (0.0-1.0)
}

// SequencerParamUpdate represents parameter updates sent to sequencer PID
type SequencerParamUpdate struct {
	Timestamp           time.Time
	NewKp               float64 // Updated proportional gain
	NewKi               float64 // Updated integral gain
	NewKd               float64 // Updated derivative gain
	NewTargetUtil       float64 // Updated target utilization
	NewMaxFeeChange     float64 // Updated max fee change per block
	ThrottlingActive    bool    // Whether to activate throttling
	ThrottlingIntensity float64 // Throttling intensity (0.0-1.0)
	Reason              string  // Reason for the update
}

// BatcherSlowPIDConfig holds configuration for the strategic batcher PID
type BatcherSlowPIDConfig struct {
	// Base configuration
	TargetBlockSize uint64
	BurstMultiplier float64
	InitialBaseFee  uint64
	MinBaseFee      uint64

	// Batcher-specific parameters
	DAWindowSize     int           // Number of DA metrics to consider (e.g., 10 blocks)
	UpdateFrequency  time.Duration // How often to send updates (20-60s)
	L1ResponseWindow time.Duration // How long to analyze L1 trends

	// Strategic PID parameters (slower, more stable)
	Kp float64 // Proportional gain for DA cost response
	Ki float64 // Integral gain for sustained DA pressure
	Kd float64 // Derivative gain for DA trend detection

	// DA cost management
	TargetDAUtilization float64 // Target DA utilization (0.8 = 80%)
	MaxDAUtilization    float64 // Emergency throttling threshold
	DABudgetPerHour     uint64  // Max DA cost budget per hour

	// Sequencer coordination parameters
	SequencerKpRange   [2]float64 // Min/Max Kp values for sequencer
	SequencerKiRange   [2]float64 // Min/Max Ki values for sequencer
	SequencerKdRange   [2]float64 // Min/Max Kd values for sequencer
	MaxParameterChange float64    // Max parameter change per update

	// Integral windup protection
	MaxIntegral float64
	MinIntegral float64
}

// DefaultBatcherSlowPIDConfig returns optimized defaults for strategic DA management
func DefaultBatcherSlowPIDConfig() *BatcherSlowPIDConfig {
	return &BatcherSlowPIDConfig{
		TargetBlockSize: 15_000_000,
		BurstMultiplier: 2.0,
		InitialBaseFee:  1_000_000_000,
		MinBaseFee:      0,

		// Slow strategic control
		DAWindowSize:     10,               // 10 blocks = ~2 minutes
		UpdateFrequency:  30 * time.Second, // Update every 30 seconds
		L1ResponseWindow: 5 * time.Minute,  // Analyze 5min of L1 trends

		// Tuned PID parameters for strategic control
		Kp: 2.0,   // Tuned proportional gain
		Ki: 0.306, // Tuned integral gain
		Kd: 0.3,   // Tuned derivative gain

		// DA management targets
		TargetDAUtilization: 0.75,                    // Target 75% DA utilization
		MaxDAUtilization:    0.90,                    // Emergency throttling at 90%
		DABudgetPerHour:     100_000_000_000_000_000, // 0.1 ETH per hour

		// Sequencer parameter ranges
		SequencerKpRange:   [2]float64{0.1, 2.0},   // Sequencer Kp range
		SequencerKiRange:   [2]float64{0.01, 0.5},  // Sequencer Ki range
		SequencerKdRange:   [2]float64{0.005, 0.2}, // Sequencer Kd range
		MaxParameterChange: 0.2,                    // Max 20% parameter change

		// Integral limits
		MaxIntegral: 10.0,
		MinIntegral: -10.0,
	}
}

// Implement AdjusterConfig interface
func (c *BatcherSlowPIDConfig) GetTargetBlockSize() uint64  { return c.TargetBlockSize }
func (c *BatcherSlowPIDConfig) GetBurstMultiplier() float64 { return c.BurstMultiplier }
func (c *BatcherSlowPIDConfig) GetInitialBaseFee() uint64   { return c.InitialBaseFee }
func (c *BatcherSlowPIDConfig) GetMinBaseFee() uint64       { return c.MinBaseFee }

// BatcherSlowPID implements strategic DA cost management with sequencer coordination
type BatcherSlowPID struct {
	config  *BatcherSlowPIDConfig
	blocks  []Block
	baseFee uint64

	// L1/DA data tracking
	daMetrics  []DAMetrics
	l1Trends   []float64 // L1 gas price trends
	daCostHist []uint64  // DA cost history

	// PID controller state
	integral     float64
	lastError    float64
	errorHistory []float64

	// Strategic state
	lastUpdateTime  time.Time
	sequencerParams SequencerParamUpdate // Current sequencer parameters
	daUtilAvg       float64              // Moving average DA utilization
	costPerHour     uint64               // Current cost rate
	emergencyMode   bool                 // Emergency throttling active

	// Output channel for sequencer updates
	parameterUpdates chan SequencerParamUpdate
}

// NewBatcherSlowPID creates a new batcher slow PID controller
func NewBatcherSlowPID(cfg *BatcherSlowPIDConfig) FeeAdjuster {
	// Initialize with moderate sequencer parameters
	initialParams := SequencerParamUpdate{
		Timestamp:        time.Now(),
		NewKp:            0.8, // Start with responsive but stable values
		NewKi:            0.15,
		NewKd:            0.05,
		NewTargetUtil:    1.0,  // Target 100% of target block size
		NewMaxFeeChange:  0.25, // Allow up to 25% fee changes
		ThrottlingActive: false,
		Reason:           "Initial configuration",
	}

	return &BatcherSlowPID{
		config:           cfg,
		blocks:           make([]Block, 0),
		baseFee:          cfg.InitialBaseFee,
		daMetrics:        make([]DAMetrics, 0),
		l1Trends:         make([]float64, 0),
		daCostHist:       make([]uint64, 0),
		integral:         0.0,
		lastError:        0.0,
		errorHistory:     make([]float64, 0),
		lastUpdateTime:   time.Now(),
		sequencerParams:  initialParams,
		daUtilAvg:        0.0,
		costPerHour:      0,
		emergencyMode:    false,
		parameterUpdates: make(chan SequencerParamUpdate, 10),
	}
}

// GetMaxBlockSize returns max block size (used by simulation framework)
func (bp *BatcherSlowPID) GetMaxBlockSize() uint64 {
	return CalculateMaxBlockSize(bp.config.TargetBlockSize, bp.config.BurstMultiplier)
}

// ProcessBlock processes a new block and manages L1/DA strategic decisions
func (bp *BatcherSlowPID) ProcessBlock(gasUsed uint64) {
	// Add the new block
	block := Block{
		Number:  len(bp.blocks) + 1,
		GasUsed: gasUsed,
		BaseFee: bp.baseFee,
	}
	bp.blocks = append(bp.blocks, block)

	// Simulate L1/DA metrics for this block
	daMetric := bp.simulateDAMetrics(block)
	bp.daMetrics = append(bp.daMetrics, daMetric)

	// Keep only recent DA metrics
	if len(bp.daMetrics) > bp.config.DAWindowSize {
		bp.daMetrics = bp.daMetrics[1:]
	}

	// Update base fee using standard EIP-1559 (consensus layer)
	bp.updateBaseFeeEIP1559(gasUsed)

	// Check if it's time for strategic parameter update
	if time.Since(bp.lastUpdateTime) >= bp.config.UpdateFrequency {
		bp.updateStrategicParameters()
		bp.lastUpdateTime = time.Now()
	}
}

// simulateDAMetrics creates realistic L1/DA metrics for simulation
func (bp *BatcherSlowPID) simulateDAMetrics(block Block) DAMetrics {
	// Simulate realistic L1 conditions based on block usage
	baseL1Gas := uint64(20_000_000_000) // 20 Gwei base
	utilizationFactor := float64(block.GasUsed) / float64(bp.config.TargetBlockSize)

	// L1 gas price increases with utilization (simulate network effects)
	l1GasPrice := uint64(float64(baseL1Gas) * (1.0 + utilizationFactor*0.5))

	// Blob pricing (simplified model)
	blobPrice := l1GasPrice / 16 // Blob gas is typically cheaper

	// DA usage correlates with L2 block size
	daUsage := block.GasUsed / 1000 // Rough approximation: 1KB per 1000 gas
	daCapacity := uint64(131072)    // 128KB blob capacity

	// Batch cost simulation
	batchCost := l1GasPrice * 100000 // ~100k gas to submit batch

	efficiency := math.Min(float64(daUsage)/float64(daCapacity), 1.0)

	return DAMetrics{
		Timestamp:       time.Now(),
		L1GasPrice:      l1GasPrice,
		BlobPrice:       blobPrice,
		DAUsage:         daUsage,
		DACapacity:      daCapacity,
		BatchCost:       batchCost,
		BatchEfficiency: efficiency,
	}
}

// updateBaseFeeEIP1559 updates base fee using consensus rules
func (bp *BatcherSlowPID) updateBaseFeeEIP1559(gasUsed uint64) {
	targetGas := bp.config.TargetBlockSize

	if gasUsed == targetGas {
		return
	}

	// Standard EIP-1559 formula
	gasUsedDelta := int64(gasUsed) - int64(targetGas)
	baseFeeChange := int64(bp.baseFee) * gasUsedDelta / int64(targetGas) / 8

	newBaseFee := int64(bp.baseFee) + baseFeeChange
	if newBaseFee < int64(bp.config.MinBaseFee) {
		newBaseFee = int64(bp.config.MinBaseFee)
	}

	bp.baseFee = uint64(newBaseFee)
}

// updateStrategicParameters analyzes L1/DA conditions and updates sequencer parameters
func (bp *BatcherSlowPID) updateStrategicParameters() {
	if len(bp.daMetrics) == 0 {
		return
	}

	// Calculate current DA utilization and trends
	currentDAUtil := bp.calculateCurrentDAUtilization()
	daUtilError := currentDAUtil - bp.config.TargetDAUtilization

	// Update PID state
	bp.updatePIDState(daUtilError)

	// Calculate strategic adjustments
	strategicOutput := bp.calculateStrategicOutput(daUtilError)

	// Determine new sequencer parameters based on L1/DA conditions
	newParams := bp.calculateSequencerParameters(strategicOutput, currentDAUtil)

	// Send parameter update
	bp.sendParameterUpdate(newParams)
}

// calculateCurrentDAUtilization calculates current DA utilization from recent metrics
func (bp *BatcherSlowPID) calculateCurrentDAUtilization() float64 {
	if len(bp.daMetrics) == 0 {
		return 0.0
	}

	var totalUtil float64
	for _, metric := range bp.daMetrics {
		util := float64(metric.DAUsage) / float64(metric.DACapacity)
		totalUtil += util
	}

	return totalUtil / float64(len(bp.daMetrics))
}

// updatePIDState updates the strategic PID controller state
func (bp *BatcherSlowPID) updatePIDState(error float64) {
	// Update integral with windup protection
	bp.integral += error
	bp.integral = ClampFloat64(bp.integral, bp.config.MinIntegral, bp.config.MaxIntegral)

	// Update error history
	bp.errorHistory = append(bp.errorHistory, error)
	if len(bp.errorHistory) > bp.config.DAWindowSize {
		bp.errorHistory = bp.errorHistory[1:]
	}

	bp.lastError = error
}

// calculateStrategicOutput calculates the strategic PID output
func (bp *BatcherSlowPID) calculateStrategicOutput(error float64) float64 {
	proportional := bp.config.Kp * error
	integral := bp.config.Ki * bp.integral
	derivative := bp.config.Kd * bp.calculateDerivative()

	return proportional + integral + derivative
}

// calculateDerivative calculates derivative of DA utilization error
func (bp *BatcherSlowPID) calculateDerivative() float64 {
	if len(bp.errorHistory) < 2 {
		return 0.0
	}

	return bp.errorHistory[len(bp.errorHistory)-1] - bp.errorHistory[len(bp.errorHistory)-2]
}

// calculateSequencerParameters determines optimal sequencer PID parameters
func (bp *BatcherSlowPID) calculateSequencerParameters(strategicOutput float64, currentDAUtil float64) SequencerParamUpdate {
	// Base sequencer parameters
	newKp := 0.8
	newKi := 0.15
	newKd := 0.05
	newTargetUtil := 1.0
	newMaxFeeChange := 0.25
	throttlingActive := false
	throttlingIntensity := 0.0
	reason := "Strategic adjustment"

	// Adjust based on DA pressure
	if currentDAUtil > bp.config.MaxDAUtilization {
		// Emergency mode: aggressive throttling
		bp.emergencyMode = true
		throttlingActive = true
		throttlingIntensity = math.Min(0.5, (currentDAUtil-bp.config.MaxDAUtilization)*2.0)
		newTargetUtil = 0.7 // Reduce target utilization
		newKp = 1.5         // More aggressive response
		reason = fmt.Sprintf("Emergency throttling: DA util %.2f%%", currentDAUtil*100)

	} else if currentDAUtil > bp.config.TargetDAUtilization {
		// Moderate pressure: tune for efficiency
		pressureFactor := (currentDAUtil - bp.config.TargetDAUtilization) /
			(bp.config.MaxDAUtilization - bp.config.TargetDAUtilization)

		newKp = 0.8 + (0.7 * pressureFactor)             // Increase responsiveness
		newKi = 0.15 - (0.05 * pressureFactor)           // Reduce integral action
		newMaxFeeChange = 0.25 + (0.15 * pressureFactor) // Allow larger changes
		reason = fmt.Sprintf("DA pressure adjustment: util %.2f%%", currentDAUtil*100)

	} else {
		// Low pressure: optimize for user experience
		bp.emergencyMode = false
		newKp = 0.6           // Gentler response
		newKi = 0.2           // More integral action for stability
		newMaxFeeChange = 0.2 // Limit fee volatility
		reason = fmt.Sprintf("Low DA pressure: optimizing UX, util %.2f%%", currentDAUtil*100)
	}

	// Apply parameter change limits
	maxChange := bp.config.MaxParameterChange
	newKp = bp.clampParameterChange(bp.sequencerParams.NewKp, newKp, maxChange)
	newKi = bp.clampParameterChange(bp.sequencerParams.NewKi, newKi, maxChange)
	newKd = bp.clampParameterChange(bp.sequencerParams.NewKd, newKd, maxChange)

	// Apply absolute parameter ranges
	newKp = ClampFloat64(newKp, bp.config.SequencerKpRange[0], bp.config.SequencerKpRange[1])
	newKi = ClampFloat64(newKi, bp.config.SequencerKiRange[0], bp.config.SequencerKiRange[1])
	newKd = ClampFloat64(newKd, bp.config.SequencerKdRange[0], bp.config.SequencerKdRange[1])

	return SequencerParamUpdate{
		Timestamp:           time.Now(),
		NewKp:               newKp,
		NewKi:               newKi,
		NewKd:               newKd,
		NewTargetUtil:       newTargetUtil,
		NewMaxFeeChange:     newMaxFeeChange,
		ThrottlingActive:    throttlingActive,
		ThrottlingIntensity: throttlingIntensity,
		Reason:              reason,
	}
}

// clampParameterChange limits the rate of parameter changes
func (bp *BatcherSlowPID) clampParameterChange(current, desired, maxChange float64) float64 {
	change := desired - current
	maxAbsChange := current * maxChange

	if change > maxAbsChange {
		return current + maxAbsChange
	} else if change < -maxAbsChange {
		return current - maxAbsChange
	}

	return desired
}

// sendParameterUpdate sends the parameter update to sequencer
func (bp *BatcherSlowPID) sendParameterUpdate(params SequencerParamUpdate) {
	bp.sequencerParams = params

	// Send via channel (simulates RPC call)
	select {
	case bp.parameterUpdates <- params:
		// Successfully sent
	default:
		// Channel full, skip this update
		fmt.Printf("Warning: Parameter update channel full, skipping update\n")
	}
}

// GetParameterUpdates returns the channel for receiving sequencer parameter updates
func (bp *BatcherSlowPID) GetParameterUpdates() <-chan SequencerParamUpdate {
	return bp.parameterUpdates
}

// GetCurrentState returns the current state for simulation framework
func (bp *BatcherSlowPID) GetCurrentState() State {
	var targetUtilization float64
	var burstUtilization float64

	if len(bp.blocks) > 0 && len(bp.blocks) >= bp.config.DAWindowSize {
		targetUtilization = CalculateTargetUtilization(bp.blocks, bp.config.DAWindowSize, bp.config.TargetBlockSize)
		burstUtilization = CalculateBurstUtilization(bp.blocks, bp.config.DAWindowSize, bp.GetMaxBlockSize())
	}

	// Use DA utilization as learning rate for visualization
	effectiveLearningRate := bp.daUtilAvg

	return State{
		BaseFee:           bp.baseFee,
		LearningRate:      effectiveLearningRate,
		TargetUtilization: targetUtilization,
		BurstUtilization:  burstUtilization,
	}
}

// GetBlocks returns processed blocks
func (bp *BatcherSlowPID) GetBlocks() []Block {
	blocks := make([]Block, len(bp.blocks))
	copy(blocks, bp.blocks)
	return blocks
}

// Reset resets the controller state
func (bp *BatcherSlowPID) Reset() {
	bp.blocks = bp.blocks[:0]
	bp.baseFee = bp.config.InitialBaseFee
	bp.daMetrics = bp.daMetrics[:0]
	bp.l1Trends = bp.l1Trends[:0]
	bp.daCostHist = bp.daCostHist[:0]
	bp.integral = 0.0
	bp.lastError = 0.0
	bp.errorHistory = bp.errorHistory[:0]
	bp.lastUpdateTime = time.Now()
	bp.daUtilAvg = 0.0
	bp.costPerHour = 0
	bp.emergencyMode = false
}

// GetDiagnostics returns detailed diagnostic information
func (bp *BatcherSlowPID) GetDiagnostics() map[string]interface{} {
	diagnostics := make(map[string]interface{})

	if len(bp.daMetrics) > 0 {
		latest := bp.daMetrics[len(bp.daMetrics)-1]
		diagnostics["l1_gas_price_gwei"] = float64(latest.L1GasPrice) / 1e9
		diagnostics["blob_price_gwei"] = float64(latest.BlobPrice) / 1e9
		diagnostics["da_utilization"] = latest.BatchEfficiency
		diagnostics["batch_cost_eth"] = float64(latest.BatchCost) / 1e18
	}

	diagnostics["current_sequencer_kp"] = bp.sequencerParams.NewKp
	diagnostics["current_sequencer_ki"] = bp.sequencerParams.NewKi
	diagnostics["current_sequencer_kd"] = bp.sequencerParams.NewKd
	diagnostics["throttling_active"] = bp.sequencerParams.ThrottlingActive
	diagnostics["emergency_mode"] = bp.emergencyMode
	diagnostics["last_update_reason"] = bp.sequencerParams.Reason

	return diagnostics
}
