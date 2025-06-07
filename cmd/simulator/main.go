package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/brianbland/feemarketsim/pkg/analysis"
	"github.com/brianbland/feemarketsim/pkg/blockchain"
	"github.com/brianbland/feemarketsim/pkg/config"
	"github.com/brianbland/feemarketsim/pkg/scenarios"
	"github.com/brianbland/feemarketsim/pkg/simulator"
	"github.com/brianbland/feemarketsim/pkg/visualization"
)

func main() {
	// Check for special blockchain integration commands first
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "fetch-base":
			handleFetchBase()
			return
		case "simulate-base":
			handleSimulateBase()
			return
		}
	}

	// Parse configuration
	parser := config.NewParser()
	cfg, err := parser.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if cfg.Simulation.ShowHelp {
		return
	}

	// Print configuration summary
	printConfigSummary(*cfg)

	// Initialize components
	scenarioGenerator := scenarios.NewGenerator(cfg.Simulation)
	analyzer := analysis.NewAnalyzer(*cfg)
	chartGenerator := visualization.NewGenerator()

	// Determine which scenarios to run
	var scenariosToRun []scenarios.Scenario
	if cfg.Simulation.Scenario == "all" {
		allScenarios := scenarioGenerator.GenerateAll(*cfg)
		scenariosToRun = []scenarios.Scenario{
			allScenarios["full"],
			allScenarios["empty"],
			allScenarios["stable"],
			allScenarios["mixed"],
		}
	} else {
		scenario, exists := scenarioGenerator.GetByName(cfg.Simulation.Scenario, *cfg)
		if !exists {
			fmt.Fprintf(os.Stderr, "Unknown scenario: %s\n", cfg.Simulation.Scenario)
			os.Exit(1)
		}
		scenariosToRun = []scenarios.Scenario{scenario}
	}

	// Run simulations
	for _, scenario := range scenariosToRun {
		runBasicSimulation(*cfg, scenario)

		// Generate charts if requested
		if cfg.Simulation.EnableGraphs {
			if cfg.Simulation.LogScale {
				chartGenerator.GenerateChartForScenarioWithLogScale(*cfg, scenario)
			} else {
				chartGenerator.GenerateChartForScenario(*cfg, scenario)
			}
		}
	}

	// Run detailed analysis
	var analysisResults []analysis.Result
	for _, scenario := range scenariosToRun {
		result := analyzer.RunDetailedAnalysis(scenario)
		analysisResults = append(analysisResults, result)
	}

	// Print comprehensive analysis
	analysis.PrintResults(analysisResults)

	if cfg.Simulation.EnableGraphs {
		fmt.Printf("\nVisualization files generated:\n")
		for _, scenario := range scenariosToRun {
			scaleType := "linear"
			suffix := ""
			if cfg.Simulation.LogScale {
				scaleType = "logarithmic"
				suffix = "_log"
			}
			filename := fmt.Sprintf("chart_%s%s.html", strings.ToLower(strings.ReplaceAll(scenario.Name, " ", "_")), suffix)
			fmt.Printf("  - %s (AIMD fee evolution - %s scale)\n", filename, scaleType)
		}
	}
}

// printConfigSummary prints the configuration being used
func printConfigSummary(cfg config.Config) {
	simCfg := cfg.Simulation
	adjusterCfg := cfg.Adjuster

	fmt.Printf("Running Fee Market Simulation with configuration:\n")
	fmt.Printf("  Adjuster Type: %s\n", cfg.Simulation.AdjusterType)

	// Core parameters (always shown)
	fmt.Printf("  Target Block Size: %d gas (%.1f M)\n", cfg.TargetBlockSize, float64(cfg.TargetBlockSize)/1e6)
	fmt.Printf("  Burst Multiplier: %.1fx (%.1f M gas max)\n", cfg.BurstMultiplier,
		float64(cfg.TargetBlockSize)*cfg.BurstMultiplier/1e6)
	fmt.Printf("  Initial Base Fee: %.3f Gwei\n", float64(cfg.InitialBaseFee)/1e9)
	fmt.Printf("  Min Base Fee: %.3f Gwei\n", float64(cfg.MinBaseFee)/1e9)
	if simCfg.Randomizer.GaussianNoise > 0 || simCfg.Randomizer.BurstProbability > 0 {
		fmt.Printf("  Randomizer Seed: %d\n", simCfg.Randomizer.Seed)
		if simCfg.Randomizer.GaussianNoise > 0 {
			fmt.Printf("  Gaussian Noise: %.1f%%\n", simCfg.Randomizer.GaussianNoise*100)
		}
		if simCfg.Randomizer.BurstProbability > 0 {
			fmt.Printf("  Burst Probability: %.1f%%\n", simCfg.Randomizer.BurstProbability*100)
			fmt.Printf("  Burst Duration Min: %d blocks\n", simCfg.Randomizer.BurstDurationMin)
			fmt.Printf("  Burst Duration Max: %d blocks\n", simCfg.Randomizer.BurstDurationMax)
			fmt.Printf("  Burst Intensity: %.1f\n", simCfg.Randomizer.BurstIntensity)
		}
	}

	// Algorithm-specific parameters
	switch simCfg.AdjusterType {
	case "aimd":
		fmt.Printf("  Window Size: %d blocks\n", cfg.WindowSize)
		fmt.Printf("  Gamma: %.3f\n", adjusterCfg.AIMD.Gamma)
		fmt.Printf("  Learning Rate Range: %.6f - %.6f\n", adjusterCfg.AIMD.MinLearningRate, adjusterCfg.AIMD.MaxLearningRate)
		fmt.Printf("  Alpha: %.6f, Beta: %.6f\n", adjusterCfg.AIMD.Alpha, adjusterCfg.AIMD.Beta)
		fmt.Printf("  Delta: %.9f\n", adjusterCfg.AIMD.Delta)
		fmt.Printf("  Initial Learning Rate: %.6f\n", adjusterCfg.AIMD.InitialLearningRate)

	case "eip1559", "eip-1559":
		fmt.Printf("  Max Fee Change: %.1f%% per block\n", adjusterCfg.EIP1559.MaxFeeChange*100)

	case "pid":
		fmt.Printf("  Window Size: %d blocks\n", cfg.WindowSize)
		fmt.Printf("  PID Gains: Kp=%.3f, Ki=%.3f, Kd=%.3f\n", adjusterCfg.PID.Kp, adjusterCfg.PID.Ki, adjusterCfg.PID.Kd)
		fmt.Printf("  Max Fee Change: %.1f%% per block\n", adjusterCfg.PID.MaxFeeChange*100)
		fmt.Printf("  Integral Limits: %.1f to %.1f\n", adjusterCfg.PID.MinIntegral, adjusterCfg.PID.MaxIntegral)
	}

	fmt.Printf("  Scenario: %s\n", simCfg.Scenario)
	fmt.Printf("  Generate Charts: %t\n", simCfg.EnableGraphs)
	if simCfg.EnableGraphs {
		scaleType := "linear"
		if simCfg.LogScale {
			scaleType = "logarithmic"
		}
		fmt.Printf("  Chart Scale: %s\n", scaleType)
	}
	fmt.Println()
}

// runBasicSimulation runs a basic simulation and prints results
func runBasicSimulation(cfg config.Config, scenario scenarios.Scenario) {
	simCfg := cfg.Simulation

	fmt.Printf("\n=== Simulation: %s ===\n", scenario.Name)
	fmt.Printf("Description: %s\n", scenario.Description)
	fmt.Printf("Adjuster Type: %s\n", simCfg.AdjusterType)
	fmt.Printf("Burst Capacity: %.1fx target (%.0f M gas max)\n",
		cfg.BurstMultiplier, float64(cfg.TargetBlockSize)*cfg.BurstMultiplier/1e6)
	fmt.Println()

	// Parse adjuster type and create adjuster
	adjusterType, err := simulator.ParseAdjusterType(simCfg.AdjusterType)
	if err != nil {
		fmt.Printf("Error: Invalid adjuster type: %v\n", err)
		return
	}

	factory := simulator.NewAdjusterFactory()
	adjuster, err := factory.CreateAdjusterWithConfigs(adjusterType, &cfg)

	if err != nil {
		fmt.Printf("Error: Failed to create adjuster: %v\n", err)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Block\tGas Used\tTarget %\tBurst %\tBase Fee\tLearning Rate\tTarget Util")

	for i, gasUsed := range scenario.Blocks {
		adjuster.ProcessBlock(gasUsed)
		state := adjuster.GetCurrentState()

		targetPercent := float64(gasUsed) / float64(cfg.TargetBlockSize) * 100
		burstPercent := state.BurstUtilization * 100

		fmt.Fprintf(w, "%d\t%d\t%.1f%%\t%.1f%%\t%d\t%.6f\t%.3f\n",
			i+1, gasUsed, targetPercent, burstPercent, state.BaseFee,
			state.LearningRate, state.TargetUtilization)
	}
	w.Flush()
}

// handleFetchBase handles blockchain data fetching
func handleFetchBase() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: feemarketsim fetch-base <start_block> <end_block> <output_file>")
		fmt.Println("Example: feemarketsim fetch-base 12000000 12000100 base_data.json")
		return
	}

	startBlock, err := strconv.ParseUint(os.Args[2], 10, 64)
	if err != nil {
		fmt.Printf("Invalid start block: %v\n", err)
		return
	}

	endBlock, err := strconv.ParseUint(os.Args[3], 10, 64)
	if err != nil {
		fmt.Printf("Invalid end block: %v\n", err)
		return
	}

	filename := os.Args[4]

	if startBlock >= endBlock {
		fmt.Printf("Error: start block (%d) must be less than end block (%d)\n", startBlock, endBlock)
		return
	}

	blockCount := endBlock - startBlock + 1
	if blockCount > 10000 {
		fmt.Printf("Warning: Fetching %d blocks may take a long time and consume significant resources.\n", blockCount)
		fmt.Printf("Consider using smaller ranges (e.g., 100-1000 blocks) for testing.\n")
		fmt.Print("Continue? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Fetch cancelled.")
			return
		}
	}

	// Create blockchain client and fetcher
	client := blockchain.NewBaseRPCClient()
	fetchOptions := blockchain.DefaultFetchOptions(startBlock, endBlock)
	fetcher := blockchain.NewBlockFetcher(client, fetchOptions)

	// Set up context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour*2) // 2 hour timeout
	defer cancel()

	// Progress callback
	progressCallback := func(progress blockchain.FetchProgress) {
		if progress.Completed > 0 {
			elapsed := time.Since(progress.StartTime)
			rate := float64(progress.Completed) / elapsed.Seconds()
			fmt.Printf("Progress: %d/%d completed (%.1f%%), %.1f blocks/sec\n",
				progress.Completed, progress.Total,
				float64(progress.Completed)/float64(progress.Total)*100, rate)
		}
	}

	// Fetch the data
	fmt.Printf("Starting blockchain data fetch...\n")
	dataset, err := fetcher.FetchRange(ctx, progressCallback)
	if err != nil {
		fmt.Printf("Failed to fetch blockchain data: %v\n", err)
		return
	}

	// Save to file
	if err := blockchain.SaveDataSetToFile(dataset, filename); err != nil {
		fmt.Printf("Failed to save dataset: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Successfully fetched and saved %d blocks to %s\n", len(dataset.Blocks), filename)
	fmt.Printf("Dataset contains:\n")
	fmt.Printf("  - Blocks: %d to %d\n", dataset.StartBlock, dataset.EndBlock)
	fmt.Printf("  - Initial Base Fee: %.3f Gwei\n", float64(dataset.InitialBaseFee)/1e9)
	fmt.Printf("  - Initial Gas Limit: %.1f M gas\n", float64(dataset.InitialGasLimit)/1e6)

	// Count total transactions
	totalTx := 0
	for _, block := range dataset.Blocks {
		totalTx += len(block.Transactions)
	}
	fmt.Printf("  - Total Transactions: %d\n", totalTx)
}

// handleSimulateBase handles blockchain simulation
func handleSimulateBase() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: feemarketsim simulate-base <data_file> [-graph] [other flags...]")
		fmt.Println("Example: feemarketsim simulate-base base_data.json -graph")
		fmt.Println("Example: feemarketsim simulate-base base_data.json -graph -gamma=0.1 -alpha=0.02")
		return
	}

	filename := os.Args[2]

	// Parse remaining flags
	parser := config.NewParser()
	cfg, err := parser.Parse(os.Args[3:])
	if err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		return
	}

	// Load the dataset
	fmt.Printf("Loading blockchain dataset from %s...\n", filename)
	dataset, err := blockchain.LoadDataSetFromFile(filename)
	if err != nil {
		fmt.Printf("Failed to load dataset: %v\n", err)
		return
	}

	// Validate the dataset
	if err := blockchain.ValidateDataSet(dataset); err != nil {
		fmt.Printf("Dataset validation failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Loaded valid dataset with %d blocks\n", len(dataset.Blocks))

	// Parse adjuster type
	adjusterType, err := simulator.ParseAdjusterType(cfg.Simulation.AdjusterType)
	if err != nil {
		fmt.Printf("Invalid adjuster type: %v\n", err)
		return
	}

	// Create blockchain simulator and chart generator
	blockchainSim := blockchain.NewSimulator(*cfg, adjusterType)
	chartGenerator := visualization.NewGenerator()

	// Run simulation against the dataset
	simResult, analysisResult, err := blockchainSim.SimulateAgainstDataSetWithOptions(dataset, cfg.Simulation.EnableGraphs)
	if err != nil {
		fmt.Printf("Simulation failed: %v\n", err)
		return
	}

	// Print results
	blockchain.PrintSimulationResults(simResult, analysisResult)

	// Print comparison with actual Base fees
	blockchainSim.CompareWithActualBaseFees(dataset, simResult)

	// Generate charts if requested
	if cfg.Simulation.EnableGraphs {
		filename := fmt.Sprintf("base_comparison_%d_%d.html", dataset.StartBlock, dataset.EndBlock)
		if cfg.Simulation.LogScale {
			chartGenerator.GenerateBaseComparisonChartWithLogScale(*cfg, dataset, simResult, filename)
		} else {
			chartGenerator.GenerateBaseComparisonChart(*cfg, dataset, simResult, filename)
		}

		fmt.Printf("\nVisualization files generated:\n")
		scaleType := "linear"
		if cfg.Simulation.LogScale {
			scaleType = "logarithmic"
		}
		fmt.Printf("  - %s (AIMD vs Base fee comparison - %s scale)\n", filename, scaleType)
		gasFilename := fmt.Sprintf("base_comparison_%d_%d_gas.html", dataset.StartBlock, dataset.EndBlock)
		fmt.Printf("  - %s (Gas usage analysis)\n", gasFilename)
	}
}
