package blockchain

import (
	"fmt"
	"strings"

	"github.com/brianbland/feemarketsim/pkg/analysis"
	"github.com/brianbland/feemarketsim/pkg/config"
	"github.com/brianbland/feemarketsim/pkg/scenarios"
	"github.com/brianbland/feemarketsim/pkg/simulator"
)

// Simulator handles simulation against real blockchain data
type Simulator struct {
	config       config.Config
	adjusterType simulator.AdjusterType
}

// NewSimulator creates a new blockchain simulator
func NewSimulator(cfg config.Config, adjusterType simulator.AdjusterType) *Simulator {
	return &Simulator{
		config:       cfg,
		adjusterType: adjusterType,
	}
}

// SimulateAgainstDataSet runs the AIMD mechanism against real blockchain data
func (s *Simulator) SimulateAgainstDataSet(dataset *DataSet) (*SimulationResult, *analysis.Result, error) {
	return s.SimulateAgainstDataSetWithOptions(dataset, false)
}

// SimulateAgainstDataSetWithOptions runs the AIMD mechanism with optional detailed data collection
func (s *Simulator) SimulateAgainstDataSetWithOptions(dataset *DataSet, collectVisualizationData bool) (*SimulationResult, *analysis.Result, error) {
	if err := ValidateDataSet(dataset); err != nil {
		return nil, nil, fmt.Errorf("invalid dataset: %w", err)
	}

	fmt.Printf("\n=== Simulating Against Base Blockchain Data ===\n")
	fmt.Printf("Block Range: %d - %d (%d blocks)\n", dataset.StartBlock, dataset.EndBlock, len(dataset.Blocks))
	fmt.Printf("Initial Base Fee: %.3f Gwei\n", float64(dataset.InitialBaseFee)/1e9)
	fmt.Printf("Initial Gas Limit: %.1f M gas\n\n", float64(dataset.InitialGasLimit)/1e6)

	// Override config with real initial conditions
	adjustedConfig := s.config
	adjustedConfig.InitialBaseFee = dataset.InitialBaseFee
	adjustedConfig.TargetBlockSize = dataset.InitialGasLimit / 2

	// Create fee adjuster using factory
	factory := simulator.NewAdjusterFactory()
	adjuster, err := factory.CreateAdjusterWithConfigs(s.adjusterType, &adjustedConfig)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create fee adjuster: %w", err)
	}

	var (
		totalTx   int
		droppedTx int
		baseFees  []uint64
		gasUsages []uint64
		compData  *ComparisonData
	)

	// Initialize comparison data if requested
	if collectVisualizationData {
		compData = &ComparisonData{
			BlockNumbers:       make([]float64, 0, len(dataset.Blocks)),
			ActualBaseFees:     make([]float64, 0, len(dataset.Blocks)),
			SimulatedBaseFees:  make([]float64, 0, len(dataset.Blocks)),
			DroppedPercentages: make([]float64, 0, len(dataset.Blocks)),
			ActualGasUsages:    make([]float64, 0, len(dataset.Blocks)),
			EffectiveGasUsages: make([]float64, 0, len(dataset.Blocks)),
			LearningRates:      make([]float64, 0, len(dataset.Blocks)),
		}
	}

	// Simulate each block
	for i, block := range dataset.Blocks {
		currentBaseFee := adjuster.GetCurrentState().BaseFee

		// Calculate transaction dropping and effective gas usage
		effectiveGasUsed, blockDropped := s.calculateTransactionDropping(block, currentBaseFee)

		totalTx += len(block.Transactions)
		droppedTx += blockDropped

		// Process block with effective gas usage
		adjuster.ProcessBlock(effectiveGasUsed)
		state := adjuster.GetCurrentState()

		baseFees = append(baseFees, state.BaseFee)
		gasUsages = append(gasUsages, effectiveGasUsed)

		// Collect visualization data if requested
		if collectVisualizationData {
			droppedPercentage := 0.0
			if len(block.Transactions) > 0 {
				droppedPercentage = float64(blockDropped) / float64(len(block.Transactions)) * 100
			}

			compData.BlockNumbers = append(compData.BlockNumbers, float64(i+1))
			compData.ActualBaseFees = append(compData.ActualBaseFees, float64(block.BaseFeePerGas)/1e9)
			compData.SimulatedBaseFees = append(compData.SimulatedBaseFees, float64(state.BaseFee)/1e9)
			compData.DroppedPercentages = append(compData.DroppedPercentages, droppedPercentage)
			compData.ActualGasUsages = append(compData.ActualGasUsages, float64(block.GasUsed)/1e6)
			compData.EffectiveGasUsages = append(compData.EffectiveGasUsages, float64(effectiveGasUsed)/1e6)
			compData.LearningRates = append(compData.LearningRates, state.LearningRate)
		}

		if i < 10 || i%50 == 0 {
			fmt.Printf("Block %d: Gas Used: %d, Base Fee: %.3f Gwei, Dropped Tx: %d\n",
				block.Number, effectiveGasUsed, float64(state.BaseFee)/1e9, blockDropped)
		}
	}

	// Calculate simulation results
	simResult := s.calculateSimulationResult(totalTx, droppedTx, baseFees, gasUsages, adjustedConfig)
	simResult.ComparisonData = compData

	// Create scenario for analysis
	scenario := scenarios.Scenario{
		Name:        "Base Blockchain Data",
		Description: fmt.Sprintf("Real data from Base blocks %d-%d", dataset.StartBlock, dataset.EndBlock),
		Blocks:      gasUsages,
	}

	// Run detailed analysis
	analyzer := analysis.NewAnalyzer(adjustedConfig)
	analysisResult := analyzer.RunDetailedAnalysis(scenario)

	return simResult, &analysisResult, nil
}

// SimulateForVisualization runs simulation specifically for chart generation
func (s *Simulator) SimulateForVisualization(dataset *DataSet) (*SimulationResult, error) {
	result, _, err := s.SimulateAgainstDataSetWithOptions(dataset, true)
	return result, err
}

// calculateTransactionDropping determines which transactions would be dropped and calculates effective gas usage
func (s *Simulator) calculateTransactionDropping(block BlockData, currentBaseFee uint64) (uint64, int) {
	var effectiveGasUsed uint64
	droppedCount := 0

	for _, tx := range block.Transactions {
		// Determine transaction's maximum fee willingness
		txMaxFee := s.getTransactionMaxFee(tx, currentBaseFee)

		if txMaxFee >= currentBaseFee {
			// Transaction would be included
			effectiveGasUsed += tx.GasUsed
		} else {
			// Transaction would be dropped
			droppedCount++
		}
	}

	return effectiveGasUsed, droppedCount
}

// getTransactionMaxFee determines the maximum fee a transaction is willing to pay
func (s *Simulator) getTransactionMaxFee(tx Transaction, currentBaseFee uint64) uint64 {
	// For EIP-1559 transactions, use maxFeePerGas
	if tx.MaxFeePerGas > 0 {
		return tx.MaxFeePerGas
	}

	// For legacy transactions, use gasPrice
	if tx.GasPrice > 0 {
		return tx.GasPrice
	}

	// If no fee info available, assume transaction would be included with buffer
	return currentBaseFee + 1_000_000_000 // Add 1 Gwei buffer
}

// calculateSimulationResult computes the final simulation metrics
func (s *Simulator) calculateSimulationResult(totalTx, droppedTx int, baseFees, gasUsages []uint64, cfg config.Config) *SimulationResult {
	droppedPercentage := 0.0
	if totalTx > 0 {
		droppedPercentage = float64(droppedTx) / float64(totalTx) * 100
	}

	avgBaseFee := s.averageUint64(baseFees)
	totalGasUsed := s.sumUint64(gasUsages)
	targetCapacity := uint64(len(gasUsages)) * cfg.TargetBlockSize
	effectiveUtilization := float64(totalGasUsed) / float64(targetCapacity)

	return &SimulationResult{
		TotalTransactions:    totalTx,
		DroppedTransactions:  droppedTx,
		DroppedPercentage:    droppedPercentage,
		AvgBaseFee:           uint64(avgBaseFee),
		MaxBaseFee:           s.maxUint64(baseFees),
		MinBaseFee:           s.minUint64(baseFees),
		TotalGasUsed:         totalGasUsed,
		EffectiveUtilization: effectiveUtilization,
	}
}

// PrintSimulationResults prints the results of blockchain simulation
func PrintSimulationResults(simResult *SimulationResult, analysisResult *analysis.Result) {
	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	fmt.Printf("BASE BLOCKCHAIN SIMULATION RESULTS\n")
	fmt.Printf(strings.Repeat("=", 80) + "\n")

	fmt.Printf("Transaction Processing:\n")
	fmt.Printf("  Total Transactions: %d\n", simResult.TotalTransactions)
	fmt.Printf("  Dropped Transactions: %d (%.2f%%)\n", simResult.DroppedTransactions, simResult.DroppedPercentage)
	fmt.Printf("  Effective Utilization: %.2f%%\n", simResult.EffectiveUtilization*100)

	fmt.Printf("\nFee Market Performance:\n")
	fmt.Printf("  Average Base Fee: %.3f Gwei\n", float64(simResult.AvgBaseFee)/1e9)
	fmt.Printf("  Fee Range: %.3f - %.3f Gwei\n",
		float64(simResult.MinBaseFee)/1e9, float64(simResult.MaxBaseFee)/1e9)
	fmt.Printf("  Total Gas Processed: %.1f M gas\n", float64(simResult.TotalGasUsed)/1e6)

	fmt.Printf("\nAIMD Mechanism Analysis:\n")
	fmt.Printf("  Final Fee vs Initial: %.2fx\n",
		float64(analysisResult.FinalBaseFee)/float64(analysisResult.InitialBaseFee))
	fmt.Printf("  Fee Volatility: %.3f Gwei\n", analysisResult.BaseFeeVolatility/1e9)
	fmt.Printf("  Average Learning Rate: %.6f\n", analysisResult.AvgLearningRate)
	fmt.Printf("  Responsiveness Score: %.3f\n", analysisResult.ResponsivenessScore)
}

// Utility functions for calculating statistics

func (s *Simulator) sumUint64(values []uint64) uint64 {
	var sum uint64
	for _, v := range values {
		sum += v
	}
	return sum
}

func (s *Simulator) averageUint64(values []uint64) float64 {
	if len(values) == 0 {
		return 0
	}
	return float64(s.sumUint64(values)) / float64(len(values))
}

func (s *Simulator) minUint64(values []uint64) uint64 {
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

func (s *Simulator) maxUint64(values []uint64) uint64 {
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

// CompareWithActualBaseFees compares simulated results with actual Base blockchain fees
func (s *Simulator) CompareWithActualBaseFees(dataset *DataSet, simResult *SimulationResult) {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("SIMULATED vs ACTUAL BASE FEES COMPARISON\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	// Calculate statistics for actual Base fees
	var actualFees []uint64
	for _, block := range dataset.Blocks {
		actualFees = append(actualFees, block.BaseFeePerGas)
	}

	actualAvg := s.averageUint64(actualFees)
	actualMin := s.minUint64(actualFees)
	actualMax := s.maxUint64(actualFees)

	fmt.Printf("Actual Base Fees (EIP-1559):\n")
	fmt.Printf("  Average: %.3f Gwei\n", actualAvg/1e9)
	fmt.Printf("  Range: %.3f - %.3f Gwei\n", float64(actualMin)/1e9, float64(actualMax)/1e9)

	fmt.Printf("\nSimulation Results:\n")
	fmt.Printf("  Average: %.3f Gwei\n", float64(simResult.AvgBaseFee)/1e9)
	fmt.Printf("  Range: %.3f - %.3f Gwei\n", float64(simResult.MinBaseFee)/1e9, float64(simResult.MaxBaseFee)/1e9)

	fmt.Printf("\nComparison:\n")
	avgRatio := float64(simResult.AvgBaseFee) / actualAvg
	fmt.Printf("  Simulated/Actual Average Ratio: %.3fx\n", avgRatio)

	if avgRatio > 1.1 {
		fmt.Printf("  → Simulated fees are %.1f%% higher than actual\n", (avgRatio-1)*100)
	} else if avgRatio < 0.9 {
		fmt.Printf("  → Simulated fees are %.1f%% lower than actual\n", (1-avgRatio)*100)
	} else {
		fmt.Printf("  → Simulated fees are comparable to actual (within 10%%)\n")
	}
}
