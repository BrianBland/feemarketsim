package analysis

import (
	"fmt"
	"math"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/brianbland/feemarketsim/pkg/config"
	"github.com/brianbland/feemarketsim/pkg/scenarios"
	"github.com/brianbland/feemarketsim/pkg/simulator"
)

// Result contains detailed analysis of a simulation run
type Result struct {
	ScenarioName           string
	TotalBlocks            int
	AvgGasUsed             float64
	AvgGasUsedPercent      float64
	AvgBlockConsumption    float64
	InitialBaseFee         uint64
	FinalBaseFee           uint64
	MinBaseFee             uint64
	MaxBaseFee             uint64
	BaseFeeVolatility      float64
	AvgLearningRate        float64
	MinLearningRate        float64
	MaxLearningRate        float64
	LearningRateVolatility float64
	TargetDeviation        float64
	ResponsivenessScore    float64
}

// Analyzer handles analysis operations
type Analyzer struct {
	config config.Config
}

// NewAnalyzer creates a new analyzer
func NewAnalyzer(cfg config.Config) *Analyzer {
	return &Analyzer{config: cfg}
}

// RunDetailedAnalysis runs a simulation and provides comprehensive analysis
func (a *Analyzer) RunDetailedAnalysis(scenario scenarios.Scenario) Result {
	adjuster := simulator.NewFeeAdjuster(a.config)

	var (
		baseFees           []uint64
		learningRates      []float64
		targetUtilizations []float64
		burstUtilizations  []float64
		gasUsages          []uint64
		targetDeviations   []float64
	)

	for _, gasUsed := range scenario.Blocks {
		adjuster.ProcessBlock(gasUsed)
		state := adjuster.GetCurrentState()

		baseFees = append(baseFees, state.BaseFee)
		learningRates = append(learningRates, state.LearningRate)
		targetUtilizations = append(targetUtilizations, state.TargetUtilization)
		burstUtilizations = append(burstUtilizations, state.BurstUtilization)
		gasUsages = append(gasUsages, gasUsed)

		// Calculate deviation from target
		deviation := math.Abs(float64(gasUsed)-float64(a.config.TargetBlockSize)) / float64(a.config.TargetBlockSize)
		targetDeviations = append(targetDeviations, deviation)
	}

	// Calculate statistics
	avgGasUsed := averageUint64(gasUsages)
	maxBlockSize := uint64(float64(a.config.TargetBlockSize) * a.config.BurstMultiplier)
	avgGasUsedPercent := avgGasUsed / float64(maxBlockSize) * 100

	// Handle case where dataset is smaller than window size
	var avgBurstUtilization float64
	if len(burstUtilizations) >= a.config.WindowSize {
		avgBurstUtilization = averageFloat64(burstUtilizations[a.config.WindowSize-1:]) // Only after window fills
	} else {
		avgBurstUtilization = averageFloat64(burstUtilizations) // Use all available data
	}

	avgLearningRate := averageFloat64(learningRates)
	avgTargetDeviation := averageFloat64(targetDeviations)

	// Calculate volatilities (standard deviation)
	baseFeeVolatility := stdDev(convertToFloat64(baseFees))
	learningRateVolatility := stdDev(learningRates)

	// Calculate responsiveness score
	responsivenessScore := a.calculateResponsiveness(gasUsages, baseFees)

	return Result{
		ScenarioName:           scenario.Name,
		TotalBlocks:            len(scenario.Blocks),
		AvgGasUsed:             avgGasUsed,
		AvgGasUsedPercent:      avgGasUsedPercent,
		AvgBlockConsumption:    avgBurstUtilization,
		InitialBaseFee:         a.config.InitialBaseFee,
		FinalBaseFee:           baseFees[len(baseFees)-1],
		MinBaseFee:             minUint64(baseFees),
		MaxBaseFee:             maxUint64(baseFees),
		BaseFeeVolatility:      baseFeeVolatility,
		AvgLearningRate:        avgLearningRate,
		MinLearningRate:        minFloat64(learningRates),
		MaxLearningRate:        maxFloat64(learningRates),
		LearningRateVolatility: learningRateVolatility,
		TargetDeviation:        avgTargetDeviation,
		ResponsivenessScore:    responsivenessScore,
	}
}

// calculateResponsiveness measures how well fees respond to demand changes
func (a *Analyzer) calculateResponsiveness(gasUsages []uint64, baseFees []uint64) float64 {
	if len(gasUsages) <= a.config.WindowSize {
		return 0
	}

	var responsiveness float64
	count := 0

	// Look at periods where demand significantly changes
	for i := a.config.WindowSize; i < len(gasUsages)-1; i++ {
		// Calculate demand change
		currentDemand := float64(gasUsages[i]) / float64(a.config.TargetBlockSize)
		prevDemand := float64(gasUsages[i-1]) / float64(a.config.TargetBlockSize)
		demandChange := math.Abs(currentDemand - prevDemand)

		// Only consider significant demand changes
		if demandChange > 0.2 { // 20% change threshold
			// Calculate fee response
			currentFee := float64(baseFees[i])
			prevFee := float64(baseFees[i-1])
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

// PrintResults prints formatted analysis results
func PrintResults(results []Result) {
	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	fmt.Printf("COMPREHENSIVE ANALYSIS SUMMARY\n")
	fmt.Printf(strings.Repeat("=", 80) + "\n")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Scenario\tAvg Gas %\tFinal Fee\tFee Range\tFee Volatility\tResponsiveness")

	for _, result := range results {
		feeChange := float64(result.FinalBaseFee) / float64(result.InitialBaseFee)
		feeRangeStr := fmt.Sprintf("%.2fx", float64(result.MaxBaseFee)/float64(result.MinBaseFee))
		volatilityStr := fmt.Sprintf("%.0f", result.BaseFeeVolatility/1e9) // Convert to Gwei
		responsivenessStr := fmt.Sprintf("%.3f", result.ResponsivenessScore)

		fmt.Fprintf(w, "%s\t%.1f%%\t%.2fx\t%s\t%s Gwei\t%s\n",
			result.ScenarioName,
			result.AvgGasUsedPercent,
			feeChange,
			feeRangeStr,
			volatilityStr,
			responsivenessStr,
		)
	}
	w.Flush()

	// Detailed breakdown for each scenario
	for _, result := range results {
		fmt.Printf("\n" + strings.Repeat("-", 60) + "\n")
		fmt.Printf("DETAILED ANALYSIS: %s\n", result.ScenarioName)
		fmt.Printf(strings.Repeat("-", 60) + "\n")

		fmt.Printf("Block Statistics:\n")
		fmt.Printf("  Total Blocks: %d\n", result.TotalBlocks)
		fmt.Printf("  Average Gas Used: %.0f (%.1f%% of burst capacity)\n", result.AvgGasUsed, result.AvgGasUsedPercent)
		fmt.Printf("  Average Burst Utilization: %.3f\n", result.AvgBlockConsumption)
		fmt.Printf("  Average Deviation from Target: %.1f%%\n", result.TargetDeviation*100)

		fmt.Printf("\nBase Fee Evolution:\n")
		fmt.Printf("  Initial: %.3f Gwei\n", float64(result.InitialBaseFee)/1e9)
		fmt.Printf("  Final: %.3f Gwei (%.2fx change)\n",
			float64(result.FinalBaseFee)/1e9,
			float64(result.FinalBaseFee)/float64(result.InitialBaseFee))
		fmt.Printf("  Range: %.3f - %.3f Gwei\n",
			float64(result.MinBaseFee)/1e9,
			float64(result.MaxBaseFee)/1e9)
		fmt.Printf("  Volatility: %.3f Gwei (std dev)\n", result.BaseFeeVolatility/1e9)

		fmt.Printf("\nLearning Rate Dynamics:\n")
		fmt.Printf("  Average: %.6f\n", result.AvgLearningRate)
		fmt.Printf("  Range: %.6f - %.6f\n", result.MinLearningRate, result.MaxLearningRate)
		fmt.Printf("  Volatility: %.6f (std dev)\n", result.LearningRateVolatility)

		fmt.Printf("\nMechanism Performance:\n")
		fmt.Printf("  Responsiveness Score: %.3f\n", result.ResponsivenessScore)
		fmt.Printf("  (Higher is more responsive to demand changes)\n")
	}
}

// Utility functions for statistics calculations

func averageUint64(values []uint64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum uint64
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

func averageFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	mean := averageFloat64(values)
	var sumSquares float64
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}

	return math.Sqrt(sumSquares / float64(len(values)-1))
}

func convertToFloat64(values []uint64) []float64 {
	result := make([]float64, len(values))
	for i, v := range values {
		result[i] = float64(v)
	}
	return result
}

func minUint64(values []uint64) uint64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func maxUint64(values []uint64) uint64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func minFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func maxFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}
