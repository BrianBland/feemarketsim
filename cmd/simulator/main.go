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
	cfg, simCfg, err := parser.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if simCfg.ShowHelp {
		return
	}

	// Print configuration summary
	printConfigSummary(*cfg, *simCfg)

	// Initialize components
	scenarioGenerator := scenarios.NewGenerator(*cfg)
	analyzer := analysis.NewAnalyzer(*cfg)
	chartGenerator := visualization.NewGenerator()

	// Determine which scenarios to run
	var scenariosToRun []scenarios.Scenario
	if simCfg.Scenario == "all" {
		allScenarios := scenarioGenerator.GenerateAll(*cfg)
		scenariosToRun = []scenarios.Scenario{
			allScenarios["full"],
			allScenarios["empty"],
			allScenarios["stable"],
			allScenarios["mixed"],
		}
	} else {
		scenario, exists := scenarioGenerator.GetByName(simCfg.Scenario, *cfg)
		if !exists {
			fmt.Fprintf(os.Stderr, "Unknown scenario: %s\n", simCfg.Scenario)
			os.Exit(1)
		}
		scenariosToRun = []scenarios.Scenario{scenario}
	}

	// Run simulations
	for _, scenario := range scenariosToRun {
		runBasicSimulation(*cfg, scenario)

		// Generate charts if requested
		if simCfg.EnableGraphs {
			chartGenerator.GenerateChartForScenario(*cfg, scenario)
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

	if simCfg.EnableGraphs {
		fmt.Printf("\nVisualization files generated:\n")
		for _, scenario := range scenariosToRun {
			filename := fmt.Sprintf("chart_%s.html", strings.ToLower(strings.ReplaceAll(scenario.Name, " ", "_")))
			fmt.Printf("  - %s (AIMD fee evolution)\n", filename)
		}
	}
}

// printConfigSummary prints the configuration being used
func printConfigSummary(cfg config.Config, simCfg config.SimulationConfig) {
	fmt.Printf("Running AIMD Fee Market Simulation with configuration:\n")
	fmt.Printf("  Target Block Size: %d gas\n", cfg.TargetBlockSize)
	fmt.Printf("  Burst Multiplier: %.1fx (%.0f M gas max)\n", cfg.BurstMultiplier,
		float64(cfg.TargetBlockSize)*cfg.BurstMultiplier/1e6)
	fmt.Printf("  Window Size: %d blocks\n", cfg.WindowSize)
	fmt.Printf("  Gamma: %.3f\n", cfg.Gamma)
	fmt.Printf("  Learning Rate Range: %.6f - %.6f\n", cfg.MinLearningRate, cfg.MaxLearningRate)
	fmt.Printf("  Alpha: %.6f, Beta: %.6f\n", cfg.Alpha, cfg.Beta)
	fmt.Printf("  Delta: %.9f\n", cfg.Delta)
	fmt.Printf("  Initial Base Fee: %.3f Gwei\n", float64(cfg.InitialBaseFee)/1e9)
	fmt.Printf("  Min Base Fee: %.3f Gwei\n", float64(cfg.MinBaseFee)/1e9)
	fmt.Printf("  Initial Learning Rate: %.6f\n", cfg.InitialLearningRate)
	fmt.Printf("  Randomness Factor: %.1f%%\n", cfg.RandomnessFactor*100)
	fmt.Printf("  Scenario: %s\n", simCfg.Scenario)
	fmt.Printf("  Generate Charts: %t\n\n", simCfg.EnableGraphs)
}

// runBasicSimulation runs a basic simulation and prints results
func runBasicSimulation(cfg config.Config, scenario scenarios.Scenario) {
	fmt.Printf("\n=== Simulation: %s ===\n", scenario.Name)
	fmt.Printf("Description: %s\n", scenario.Description)
	fmt.Printf("Burst Capacity: %.1fx target (%.0f M gas max)\n",
		cfg.BurstMultiplier, float64(cfg.TargetBlockSize)*cfg.BurstMultiplier/1e6)
	fmt.Printf("Randomness Factor: %.1f%%\n\n", cfg.RandomnessFactor*100)

	adjuster := simulator.NewFeeAdjuster(cfg)

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
	cfg, simCfg, err := parser.Parse(os.Args[3:])
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

	// Create blockchain simulator and chart generator
	blockchainSim := blockchain.NewSimulator(*cfg)
	chartGenerator := visualization.NewGenerator()

	// Run simulation against the dataset
	simResult, analysisResult, err := blockchainSim.SimulateAgainstDataSetWithOptions(dataset, simCfg.EnableGraphs)
	if err != nil {
		fmt.Printf("Simulation failed: %v\n", err)
		return
	}

	// Print results
	blockchain.PrintSimulationResults(simResult, analysisResult)

	// Print comparison with actual Base fees
	blockchainSim.CompareWithActualBaseFees(dataset, simResult)

	// Generate charts if requested
	if simCfg.EnableGraphs {
		filename := fmt.Sprintf("base_comparison_%d_%d.html", dataset.StartBlock, dataset.EndBlock)
		chartGenerator.GenerateBaseComparisonChart(*cfg, dataset, simResult, filename)

		fmt.Printf("\nVisualization files generated:\n")
		fmt.Printf("  - %s (AIMD vs Base fee comparison)\n", filename)
		gasFilename := fmt.Sprintf("base_comparison_%d_%d_gas.html", dataset.StartBlock, dataset.EndBlock)
		fmt.Printf("  - %s (Gas usage analysis)\n", gasFilename)
	}
}
