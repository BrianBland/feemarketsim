package simulator

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// SequencerFastPIDConfig holds configuration for the fast sequencer PID
type SequencerFastPIDConfig struct {
	// Base configuration
	TargetBlockSize uint64
	BurstMultiplier float64
	InitialBaseFee  uint64
	MinBaseFee      uint64

	// Fast PID parameters (more responsive than slow layer)
	Kp float64 // Proportional gain - higher for fast response
	Ki float64 // Integral gain - moderate for stability
	Kd float64 // Derivative gain - higher for prediction

	// Integral windup prevention
	MaxIntegral float64
	MinIntegral float64

	// Output limits
	MaxFeeChange float64 // Maximum fee change per block
	WindowSize   int     // Window for derivative calculation

	// Fast layer specific parameters
	UpdateFrequency     time.Duration // How often to check for parameter updates
	ResponsivenessBoost float64       // Multiplier for responsiveness during congestion
	EmergencyThreshold  float64       // Utilization threshold for emergency mode
	EmergencyMaxChange  float64       // Higher max change during emergency

	// Target utilization control
	InitialTargetUtilization float64 // Initial target (can be adjusted by slow layer)
	UtilizationTolerance     float64 // Tolerance around target before adjustment
}

// DefaultSequencerFastPIDConfig returns optimized defaults for fast consensus layer control
func DefaultSequencerFastPIDConfig() *SequencerFastPIDConfig {
	return &SequencerFastPIDConfig{
		TargetBlockSize: 15_000_000,
		BurstMultiplier: 2.0,
		InitialBaseFee:  1_000_000_000,
		MinBaseFee:      0,

		// Aggressive PID tuning for fast response
		Kp: 0.8,  // High proportional gain for immediate response
		Ki: 0.15, // Moderate integral for sustained adjustments
		Kd: 0.25, // High derivative for predictive control

		// Integral limits
		MaxIntegral: 5.0,
		MinIntegral: -5.0,

		// Conservative output limits (will be overridden by slow layer)
		MaxFeeChange: 0.25, // 25% max change per block
		WindowSize:   3,    // Small window for fast response

		// Fast layer settings
		UpdateFrequency:     5 * time.Second, // Check for updates every 5 seconds
		ResponsivenessBoost: 1.5,             // 50% boost during high congestion
		EmergencyThreshold:  1.5,             // 150% target utilization = emergency
		EmergencyMaxChange:  0.5,             // 50% max change during emergency

		// Target utilization
		InitialTargetUtilization: 1.0,  // 100% of target block size
		UtilizationTolerance:     0.05, // 5% tolerance
	}
}

// Implement AdjusterConfig interface
func (c *SequencerFastPIDConfig) GetTargetBlockSize() uint64  { return c.TargetBlockSize }
func (c *SequencerFastPIDConfig) GetBurstMultiplier() float64 { return c.BurstMultiplier }
func (c *SequencerFastPIDConfig) GetInitialBaseFee() uint64   { return c.InitialBaseFee }
func (c *SequencerFastPIDConfig) GetMinBaseFee() uint64       { return c.MinBaseFee }

// SequencerFastPID implements fast consensus-layer fee adjustment
type SequencerFastPID struct {
	config  *SequencerFastPIDConfig
	blocks  []Block
	baseFee uint64

	// PID controller state
	integral     float64
	lastError    float64
	errorHistory []float64

	// Dynamic parameters (updated by slow layer)
	mu                  sync.RWMutex
	currentKp           float64
	currentKi           float64
	currentKd           float64
	currentTargetUtil   float64
	currentMaxFeeChange float64
	throttlingActive    bool
	throttlingIntensity float64
	lastParameterUpdate time.Time

	// Fast layer state
	emergencyMode       bool
	consecutiveHighUtil int
	consecutiveLowUtil  int
	responsivenessBoost float64

	// Parameter update channel (receives from slow layer)
	parameterUpdates chan SequencerParamUpdate
}

// NewSequencerFastPID creates a new fast sequencer PID controller
func NewSequencerFastPID(cfg *SequencerFastPIDConfig) FeeAdjuster {
	return &SequencerFastPID{
		config:              cfg,
		blocks:              make([]Block, 0),
		baseFee:             cfg.InitialBaseFee,
		integral:            0.0,
		lastError:           0.0,
		errorHistory:        make([]float64, 0),
		currentKp:           cfg.Kp,
		currentKi:           cfg.Ki,
		currentKd:           cfg.Kd,
		currentTargetUtil:   cfg.InitialTargetUtilization,
		currentMaxFeeChange: cfg.MaxFeeChange,
		throttlingActive:    false,
		throttlingIntensity: 0.0,
		lastParameterUpdate: time.Now(),
		emergencyMode:       false,
		consecutiveHighUtil: 0,
		consecutiveLowUtil:  0,
		responsivenessBoost: 1.0,
		parameterUpdates:    make(chan SequencerParamUpdate, 10),
	}
}

// GetMaxBlockSize returns max block size
func (fp *SequencerFastPID) GetMaxBlockSize() uint64 {
	return CalculateMaxBlockSize(fp.config.TargetBlockSize, fp.config.BurstMultiplier)
}

// ProcessBlock processes a new block with fast PID control
func (fp *SequencerFastPID) ProcessBlock(gasUsed uint64) {
	// Check for parameter updates from slow layer
	fp.checkParameterUpdates()

	// Add the new block
	block := Block{
		Number:  len(fp.blocks) + 1,
		GasUsed: gasUsed,
		BaseFee: fp.baseFee,
	}
	fp.blocks = append(fp.blocks, block)

	// Calculate utilization and error
	fp.mu.RLock()
	targetUtil := fp.currentTargetUtil
	fp.mu.RUnlock()

	currentUtilization := float64(gasUsed) / float64(fp.config.TargetBlockSize)
	error := currentUtilization - targetUtil

	// Update emergency mode and responsiveness
	fp.updateEmergencyMode(currentUtilization)
	fp.updateResponsiveness(currentUtilization)

	// Update PID state
	fp.updatePIDState(error)

	// Adjust base fee using fast PID control
	fp.adjustBaseFeeFastPID(error, currentUtilization)
}

// checkParameterUpdates checks for and applies parameter updates from slow layer
func (fp *SequencerFastPID) checkParameterUpdates() {
	select {
	case update := <-fp.parameterUpdates:
		fp.applyParameterUpdate(update)
	default:
		// No updates available
	}
}

// applyParameterUpdate applies parameter updates from the slow layer
func (fp *SequencerFastPID) applyParameterUpdate(update SequencerParamUpdate) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	// Apply PID gains
	fp.currentKp = update.NewKp
	fp.currentKi = update.NewKi
	fp.currentKd = update.NewKd

	// Apply target utilization
	fp.currentTargetUtil = update.NewTargetUtil

	// Apply fee change limits
	fp.currentMaxFeeChange = update.NewMaxFeeChange

	// Apply throttling settings
	fp.throttlingActive = update.ThrottlingActive
	fp.throttlingIntensity = update.ThrottlingIntensity

	fp.lastParameterUpdate = time.Now()

	// Log the parameter update
	fmt.Printf("Fast PID received parameter update: Kp=%.3f, Ki=%.3f, Kd=%.3f, TargetUtil=%.3f, Reason=%s\n",
		update.NewKp, update.NewKi, update.NewKd, update.NewTargetUtil, update.Reason)
}

// updateEmergencyMode updates emergency mode based on utilization
func (fp *SequencerFastPID) updateEmergencyMode(utilization float64) {
	threshold := fp.config.EmergencyThreshold

	if utilization > threshold {
		fp.consecutiveHighUtil++
		fp.consecutiveLowUtil = 0

		// Enter emergency mode after 2 consecutive high utilization blocks
		if fp.consecutiveHighUtil >= 2 && !fp.emergencyMode {
			fp.emergencyMode = true
			fmt.Printf("Block %d: Entering emergency mode (utilization: %.2f%%)\n",
				len(fp.blocks), utilization*100)
		}
	} else if utilization < 0.8 { // Exit emergency when utilization drops below 80%
		fp.consecutiveLowUtil++
		fp.consecutiveHighUtil = 0

		// Exit emergency mode after 3 consecutive low utilization blocks
		if fp.consecutiveLowUtil >= 3 && fp.emergencyMode {
			fp.emergencyMode = false
			fmt.Printf("Block %d: Exiting emergency mode (utilization: %.2f%%)\n",
				len(fp.blocks), utilization*100)
		}
	}
}

// updateResponsiveness adjusts responsiveness based on current conditions
func (fp *SequencerFastPID) updateResponsiveness(utilization float64) {
	// Boost responsiveness during high utilization
	if utilization > 1.2 { // 120% utilization
		fp.responsivenessBoost = fp.config.ResponsivenessBoost
	} else if utilization < 0.8 { // 80% utilization
		fp.responsivenessBoost = 0.8 // Reduce responsiveness during low utilization
	} else {
		fp.responsivenessBoost = 1.0 // Normal responsiveness
	}
}

// updatePIDState updates the PID controller state
func (fp *SequencerFastPID) updatePIDState(error float64) {
	// Update integral with windup protection
	fp.integral += error
	fp.integral = ClampFloat64(fp.integral, fp.config.MinIntegral, fp.config.MaxIntegral)

	// Update error history
	fp.errorHistory = append(fp.errorHistory, error)
	if len(fp.errorHistory) > fp.config.WindowSize {
		fp.errorHistory = fp.errorHistory[1:]
	}

	fp.lastError = error
}

// calculateDerivative calculates the derivative term for fast response
func (fp *SequencerFastPID) calculateDerivative() float64 {
	if len(fp.errorHistory) < 2 {
		return 0.0
	}

	// Use simple derivative for fast response
	return fp.errorHistory[len(fp.errorHistory)-1] - fp.errorHistory[len(fp.errorHistory)-2]
}

// adjustBaseFeeFastPID adjusts base fee using fast PID control
func (fp *SequencerFastPID) adjustBaseFeeFastPID(error float64, utilization float64) {
	fp.mu.RLock()
	kp := fp.currentKp
	ki := fp.currentKi
	kd := fp.currentKd
	maxChange := fp.currentMaxFeeChange
	throttling := fp.throttlingActive
	throttlingIntensity := fp.throttlingIntensity
	fp.mu.RUnlock()

	// Apply responsiveness boost to gains
	kp *= fp.responsivenessBoost
	ki *= fp.responsivenessBoost
	kd *= fp.responsivenessBoost

	// Calculate PID terms
	proportional := kp * error
	integral := ki * fp.integral
	derivative := kd * fp.calculateDerivative()

	// Calculate control output
	controlOutput := proportional + integral + derivative

	// Apply emergency mode higher limits
	if fp.emergencyMode {
		maxChange = math.Max(maxChange, fp.config.EmergencyMaxChange)
	}

	// Apply throttling if active
	if throttling {
		// Reduce max change when throttling is active
		maxChange *= (1.0 - throttlingIntensity*0.5)
		// Bias toward fee increases during throttling
		if controlOutput < 0 {
			controlOutput *= (1.0 - throttlingIntensity*0.3)
		}
	}

	// Limit control output
	controlOutput = ClampFloat64(controlOutput, -maxChange, maxChange)

	// Apply to base fee
	newBaseFee := float64(fp.baseFee) * (1.0 + controlOutput)

	// Ensure minimum base fee
	if newBaseFee < float64(fp.config.MinBaseFee) {
		newBaseFee = float64(fp.config.MinBaseFee)
	}

	fp.baseFee = uint64(newBaseFee)
}

// GetCurrentState returns current state
func (fp *SequencerFastPID) GetCurrentState() State {
	var targetUtilization float64
	var burstUtilization float64

	if len(fp.blocks) > 0 {
		windowSize := fp.config.WindowSize
		if windowSize > len(fp.blocks) {
			windowSize = len(fp.blocks)
		}

		targetUtilization = CalculateTargetUtilization(fp.blocks, windowSize, fp.config.TargetBlockSize)
		burstUtilization = CalculateBurstUtilization(fp.blocks, windowSize, fp.GetMaxBlockSize())
	}

	// Calculate effective learning rate based on recent PID activity
	effectiveLearningRate := fp.calculateEffectiveLearningRate()

	return State{
		BaseFee:           fp.baseFee,
		LearningRate:      effectiveLearningRate,
		TargetUtilization: targetUtilization,
		BurstUtilization:  burstUtilization,
	}
}

// calculateEffectiveLearningRate calculates effective learning rate based on PID gains
func (fp *SequencerFastPID) calculateEffectiveLearningRate() float64 {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	// Combine PID gains into an effective learning rate measure
	// This gives a sense of how aggressively the system is adjusting
	baseRate := (fp.currentKp + fp.currentKi + fp.currentKd) / 3.0

	// Apply emergency and responsiveness multipliers
	if fp.emergencyMode {
		baseRate *= 1.5
	}
	baseRate *= fp.responsivenessBoost

	return baseRate
}

// GetBlocks returns copy of processed blocks
func (fp *SequencerFastPID) GetBlocks() []Block {
	blocks := make([]Block, len(fp.blocks))
	copy(blocks, fp.blocks)
	return blocks
}

// Reset resets the controller to initial state
func (fp *SequencerFastPID) Reset() {
	fp.blocks = make([]Block, 0)
	fp.baseFee = fp.config.InitialBaseFee
	fp.integral = 0.0
	fp.lastError = 0.0
	fp.errorHistory = make([]float64, 0)

	fp.mu.Lock()
	fp.currentKp = fp.config.Kp
	fp.currentKi = fp.config.Ki
	fp.currentKd = fp.config.Kd
	fp.currentTargetUtil = fp.config.InitialTargetUtilization
	fp.currentMaxFeeChange = fp.config.MaxFeeChange
	fp.throttlingActive = false
	fp.throttlingIntensity = 0.0
	fp.mu.Unlock()

	fp.emergencyMode = false
	fp.consecutiveHighUtil = 0
	fp.consecutiveLowUtil = 0
	fp.responsivenessBoost = 1.0
	fp.lastParameterUpdate = time.Now()
}

// SendParameterUpdate sends parameter update to this fast PID (used by slow layer)
func (fp *SequencerFastPID) SendParameterUpdate(update SequencerParamUpdate) {
	select {
	case fp.parameterUpdates <- update:
		// Successfully sent
	default:
		// Channel full, drop oldest update
		select {
		case <-fp.parameterUpdates:
			fp.parameterUpdates <- update
		default:
			// Still full, just drop the update
		}
	}
}

// GetDiagnostics returns detailed diagnostic information
func (fp *SequencerFastPID) GetDiagnostics() map[string]interface{} {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	return map[string]interface{}{
		"current_kp":             fp.currentKp,
		"current_ki":             fp.currentKi,
		"current_kd":             fp.currentKd,
		"current_target_util":    fp.currentTargetUtil,
		"current_max_fee_change": fp.currentMaxFeeChange,
		"throttling_active":      fp.throttlingActive,
		"throttling_intensity":   fp.throttlingIntensity,
		"emergency_mode":         fp.emergencyMode,
		"responsiveness_boost":   fp.responsivenessBoost,
		"integral_term":          fp.integral,
		"last_error":             fp.lastError,
		"consecutive_high_util":  fp.consecutiveHighUtil,
		"consecutive_low_util":   fp.consecutiveLowUtil,
		"last_parameter_update":  fp.lastParameterUpdate,
	}
}
