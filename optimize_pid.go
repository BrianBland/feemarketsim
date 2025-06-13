package main

import (
	"fmt"
	"math"
	"time"

	"github.com/brianbland/feemarketsim/pkg/simulator"
)

// PerformanceMetrics holds the evaluation metrics for a parameter set
type PerformanceMetrics struct {
	AvgGasUsedPercent   float64
	FeeVolatility       float64
	ResponsivenessScore float64
	FinalFeeChange      float64
	FeeRange            float64
	ParameterSet        map[string]float64
}

// ParameterRange defines the range for a parameter
type ParameterRange struct {
	Min  float64
	Max  float64
	Step float64
}

// OptimizationConfig holds the parameter ranges to optimize
type OptimizationConfig struct {
	FastLayer struct {
		Kp                  ParameterRange
		Ki                  ParameterRange
		Kd                  ParameterRange
		WindowSize          []int
		MaxFeeChange        ParameterRange
		ResponsivenessBoost ParameterRange
		EmergencyThreshold  ParameterRange
	}
	SlowLayer struct {
		Kp                  ParameterRange
		Ki                  ParameterRange
		Kd                  ParameterRange
		DAWindowSize        []int
		TargetDAUtilization ParameterRange
		MaxDAUtilization    ParameterRange
	}
}

// DefaultOptimizationConfig returns default parameter ranges for optimization
func DefaultOptimizationConfig() *OptimizationConfig {
	cfg := &OptimizationConfig{}

	// Fast layer parameter ranges
	cfg.FastLayer.Kp = ParameterRange{0.1, 2.0, 0.2}
	cfg.FastLayer.Ki = ParameterRange{0.01, 0.5, 0.05}
	cfg.FastLayer.Kd = ParameterRange{0.01, 0.3, 0.03}
	cfg.FastLayer.WindowSize = []int{3, 5, 7}
	cfg.FastLayer.MaxFeeChange = ParameterRange{0.1, 0.5, 0.1}
	cfg.FastLayer.ResponsivenessBoost = ParameterRange{1.0, 2.0, 0.2}
	cfg.FastLayer.EmergencyThreshold = ParameterRange{1.2, 1.8, 0.2}

	// Slow layer parameter ranges
	cfg.SlowLayer.Kp = ParameterRange{0.1, 1.0, 0.1}
	cfg.SlowLayer.Ki = ParameterRange{0.01, 0.2, 0.02}
	cfg.SlowLayer.Kd = ParameterRange{0.01, 0.2, 0.02}
	cfg.SlowLayer.DAWindowSize = []int{5, 10, 15}
	cfg.SlowLayer.TargetDAUtilization = ParameterRange{0.6, 0.9, 0.1}
	cfg.SlowLayer.MaxDAUtilization = ParameterRange{0.8, 1.0, 0.05}

	return cfg
}

// TuningConstraints defines the bounds for PID parameters
type TuningConstraints struct {
	MaxKp float64
	MaxKi float64
	MaxKd float64
	MinKp float64
	MinKi float64
	MinKd float64
}

// DefaultTuningConstraints returns safe bounds for PID parameters
func DefaultTuningConstraints() *TuningConstraints {
	return &TuningConstraints{
		MaxKp: 2.0,
		MaxKi: 0.5,
		MaxKd: 0.3,
		MinKp: 0.1,
		MinKi: 0.01,
		MinKd: 0.01,
	}
}

// findCriticalGain finds the critical gain (Kp) that causes oscillation
func findCriticalGain(adjuster simulator.FeeAdjuster, targetBlockSize uint64) float64 {
	kp := 0.1
	step := 0.1
	maxKp := 5.0
	minOscillations := 2 // Reduced to find oscillation faster
	maxBlocks := 100     // Increased to get better oscillation data

	for kp < maxKp {
		// Reset adjuster
		adjuster.Reset()

		// Run simulation with current Kp
		var lastFee uint64
		var oscillations int
		var lastDirection int // 1 for increasing, -1 for decreasing
		var feeChanges []float64

		// Simulate with increasing load to force oscillation
		for i := 0; i < maxBlocks; i++ {
			// Simulate increasing load pattern
			loadFactor := 0.5 + 0.5*math.Sin(float64(i)/10.0)
			gasUsed := uint64(float64(targetBlockSize) * loadFactor)
			adjuster.ProcessBlock(gasUsed)

			// Get current fee
			state := adjuster.GetCurrentState()
			currentFee := state.BaseFee

			// Detect oscillation
			if lastFee != 0 {
				// Calculate fee change ratio
				change := float64(currentFee) / float64(lastFee)
				feeChanges = append(feeChanges, change)

				// Detect direction change with threshold
				if change > 1.01 { // 1% threshold
					if lastDirection == -1 {
						oscillations++
					}
					lastDirection = 1
				} else if change < 0.99 { // 1% threshold
					if lastDirection == 1 {
						oscillations++
					}
					lastDirection = -1
				}
			}

			lastFee = currentFee
		}

		// Check if we found critical gain
		if oscillations >= minOscillations {
			// Calculate average period
			var totalPeriod float64
			var periodCount int
			for i := 1; i < len(feeChanges); i++ {
				if (feeChanges[i] > 1.01 && feeChanges[i-1] < 0.99) ||
					(feeChanges[i] < 0.99 && feeChanges[i-1] > 1.01) {
					totalPeriod += float64(i)
					periodCount++
				}
			}

			if periodCount > 0 {
				avgPeriod := totalPeriod / float64(periodCount)
				fmt.Printf("Found critical Kp: %.3f with period: %.1f blocks\n", kp, avgPeriod)
				return kp
			}
		}

		kp += step
	}

	return kp // Return last tested Kp if no oscillation found
}

// tunePIDParameters tunes PID parameters using Ziegler-Nichols method with constraints
func tunePIDParameters(adjuster simulator.FeeAdjuster, targetBlockSize uint64, constraints *TuningConstraints) (float64, float64, float64) {
	// Find critical gain
	criticalKp := findCriticalGain(adjuster, targetBlockSize)
	fmt.Printf("Found critical Kp: %.3f\n", criticalKp)

	// Apply Ziegler-Nichols rules with constraints
	// P = 0.6 * Kc
	kp := math.Min(0.6*criticalKp, constraints.MaxKp)
	kp = math.Max(kp, constraints.MinKp)

	// I = 1.2 * Kc / Tu (where Tu is the oscillation period)
	// Using a more conservative period estimate
	period := 20.0 // Reduced from 50 to be more aggressive
	ki := math.Min(1.2*criticalKp/period, constraints.MaxKi)
	ki = math.Max(ki, constraints.MinKi)

	// D = 0.075 * Kc * Tu
	kd := math.Min(0.075*criticalKp*period, constraints.MaxKd)
	kd = math.Max(kd, constraints.MinKd)

	return kp, ki, kd
}

// evaluateTunedParameters evaluates the tuned parameters
func evaluateTunedParameters(adjuster simulator.FeeAdjuster, kp, ki, kd float64, targetBlockSize uint64) *PerformanceMetrics {
	// Reset adjuster
	adjuster.Reset()

	// Run simulation with mixed traffic pattern
	for i := 0; i < 240; i++ {
		// Simulate mixed traffic pattern with more variation
		loadFactor := 0.3 + 0.7*math.Sin(float64(i)/10.0) // Increased amplitude
		gasUsed := uint64(float64(targetBlockSize) * loadFactor)
		adjuster.ProcessBlock(gasUsed)
	}

	// Calculate metrics
	blocks := adjuster.GetBlocks()
	var totalGasUsed float64
	var maxFee, minFee uint64 = 0, math.MaxUint64
	var feeChanges []float64

	for _, block := range blocks {
		totalGasUsed += float64(block.GasUsed)
		if block.BaseFee > maxFee {
			maxFee = block.BaseFee
		}
		if block.BaseFee < minFee {
			minFee = block.BaseFee
		}
		if len(feeChanges) > 0 {
			change := float64(block.BaseFee) / float64(blocks[len(feeChanges)-1].BaseFee)
			feeChanges = append(feeChanges, change)
		}
	}

	avgGasUsed := totalGasUsed / float64(len(blocks))
	avgGasUsedPercent := avgGasUsed / float64(targetBlockSize) * 100

	// Calculate fee volatility
	var sum, sumSq float64
	for _, change := range feeChanges {
		sum += change
		sumSq += change * change
	}
	mean := sum / float64(len(feeChanges))
	variance := (sumSq / float64(len(feeChanges))) - (mean * mean)
	feeVolatility := math.Sqrt(variance)

	// Calculate responsiveness score
	responsivenessScore := calculateResponsivenessScore(blocks, targetBlockSize)

	// Calculate final fee change and range
	finalFeeChange := float64(blocks[len(blocks)-1].BaseFee) / float64(adjuster.GetCurrentState().BaseFee)
	feeRange := float64(maxFee) / float64(minFee)

	// Handle edge cases
	if math.IsNaN(feeVolatility) || math.IsInf(feeVolatility, 0) {
		feeVolatility = 0.0
	}
	if math.IsNaN(finalFeeChange) || math.IsInf(finalFeeChange, 0) {
		finalFeeChange = 1.0
	}
	if math.IsNaN(feeRange) || math.IsInf(feeRange, 0) {
		feeRange = 1.0
	}

	return &PerformanceMetrics{
		AvgGasUsedPercent:   avgGasUsedPercent,
		FeeVolatility:       feeVolatility,
		ResponsivenessScore: responsivenessScore,
		FinalFeeChange:      finalFeeChange,
		FeeRange:            feeRange,
		ParameterSet: map[string]float64{
			"kp": kp,
			"ki": ki,
			"kd": kd,
		},
	}
}

// calculateResponsivenessScore calculates how well the system responds to demand changes
func calculateResponsivenessScore(blocks []simulator.Block, targetBlockSize uint64) float64 {
	if len(blocks) < 2 {
		return 0
	}

	var responsiveness float64
	count := 0

	for i := 1; i < len(blocks); i++ {
		// Calculate demand change
		currentDemand := float64(blocks[i].GasUsed) / float64(targetBlockSize)
		prevDemand := float64(blocks[i-1].GasUsed) / float64(targetBlockSize)
		demandChange := math.Abs(currentDemand - prevDemand)

		// Only consider significant demand changes
		if demandChange > 0.2 { // 20% change threshold
			// Calculate fee response
			currentFee := float64(blocks[i].BaseFee)
			prevFee := float64(blocks[i-1].BaseFee)
			feeResponse := math.Abs((currentFee - prevFee) / prevFee)

			// Responsiveness is fee response per unit of demand change
			if demandChange > 0 {
				responsiveness += feeResponse / demandChange
				count++
			}
		}
	}

	if count == 0 {
		return 0
	}

	return responsiveness / float64(count)
}

func main() {
	// Create default configurations
	fastConfig := simulator.DefaultSequencerFastPIDConfig()
	slowConfig := simulator.DefaultBatcherSlowPIDConfig()
	hierarchicalConfig := simulator.DefaultHierarchicalPIDConfig()
	hierarchicalConfig.FastLayerConfig = fastConfig
	hierarchicalConfig.SlowLayerConfig = slowConfig

	// Create adjuster
	adjuster := simulator.NewHierarchicalPID(hierarchicalConfig)

	// Get constraints
	constraints := DefaultTuningConstraints()

	fmt.Println("Starting PID parameter tuning...")
	startTime := time.Now()

	// Tune fast layer
	fmt.Println("\nTuning Fast Layer:")
	fastKp, fastKi, fastKd := tunePIDParameters(adjuster, fastConfig.TargetBlockSize, constraints)
	fastMetrics := evaluateTunedParameters(adjuster, fastKp, fastKi, fastKd, fastConfig.TargetBlockSize)

	// Tune slow layer
	fmt.Println("\nTuning Slow Layer:")
	slowKp, slowKi, slowKd := tunePIDParameters(adjuster, slowConfig.TargetBlockSize, constraints)
	slowMetrics := evaluateTunedParameters(adjuster, slowKp, slowKi, slowKd, slowConfig.TargetBlockSize)

	duration := time.Since(startTime)

	// Print results
	fmt.Printf("\nTuning completed in %v\n", duration)
	fmt.Println("\nFast Layer Results:")
	fmt.Printf("  Kp: %.4f\n", fastKp)
	fmt.Printf("  Ki: %.4f\n", fastKi)
	fmt.Printf("  Kd: %.4f\n", fastKd)
	fmt.Printf("  Gas Used: %.1f%%\n", fastMetrics.AvgGasUsedPercent)
	fmt.Printf("  Fee Volatility: %.4f\n", fastMetrics.FeeVolatility)
	fmt.Printf("  Responsiveness: %.4f\n", fastMetrics.ResponsivenessScore)
	fmt.Printf("  Final Fee Change: %.2fx\n", fastMetrics.FinalFeeChange)
	fmt.Printf("  Fee Range: %.2fx\n", fastMetrics.FeeRange)

	fmt.Println("\nSlow Layer Results:")
	fmt.Printf("  Kp: %.4f\n", slowKp)
	fmt.Printf("  Ki: %.4f\n", slowKi)
	fmt.Printf("  Kd: %.4f\n", slowKd)
	fmt.Printf("  Gas Used: %.1f%%\n", slowMetrics.AvgGasUsedPercent)
	fmt.Printf("  Fee Volatility: %.4f\n", slowMetrics.FeeVolatility)
	fmt.Printf("  Responsiveness: %.4f\n", slowMetrics.ResponsivenessScore)
	fmt.Printf("  Final Fee Change: %.2fx\n", slowMetrics.FinalFeeChange)
	fmt.Printf("  Fee Range: %.2fx\n", slowMetrics.FeeRange)
}
