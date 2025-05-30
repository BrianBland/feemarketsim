package visualization

import (
	"os"
	"strings"
	"testing"

	"github.com/brianbland/feemarketsim/pkg/blockchain"
	"github.com/brianbland/feemarketsim/pkg/config"
	"github.com/brianbland/feemarketsim/pkg/scenarios"
)

func TestNewGenerator(t *testing.T) {
	generator := NewGenerator()
	if generator == nil {
		t.Fatal("NewGenerator() returned nil")
	}

	// Verify it implements the interface
	var _ ChartGenerator = generator
}

func TestGenerateAIMDChart(t *testing.T) {
	// Create test configuration
	cfg := config.Config{
		TargetBlockSize:     15000000,
		InitialBaseFee:      1000000000,
		MinBaseFee:          1,
		InitialLearningRate: 0.1,
		MinLearningRate:     0.001,
		MaxLearningRate:     0.5,
		Alpha:               0.01,
		Beta:                0.9,
		Delta:               0.000001,
		Gamma:               0.2,
		BurstMultiplier:     2.0,
		WindowSize:          10,
		RandomnessFactor:    0.0,
	}

	// Create test scenario
	scenario := scenarios.Scenario{
		Name:        "Test Scenario",
		Description: "Test scenario for chart generation",
		Blocks:      []uint64{15000000, 16000000, 14000000, 15500000, 14500000},
	}

	generator := NewGenerator()
	testFile := "test_chart.html"

	// Clean up any existing test file
	defer os.Remove(testFile)

	err := generator.GenerateAIMDChart(cfg, scenario, testFile)
	if err != nil {
		t.Fatalf("GenerateAIMDChart failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("Chart file was not created")
	}
}

func TestGenerateBaseComparisonChart(t *testing.T) {
	// Create test configuration
	cfg := config.Config{
		TargetBlockSize:     15000000,
		InitialBaseFee:      1000000000,
		MinBaseFee:          1,
		InitialLearningRate: 0.1,
		MinLearningRate:     0.001,
		MaxLearningRate:     0.5,
		Alpha:               0.01,
		Beta:                0.9,
		Delta:               0.000001,
		Gamma:               0.2,
		BurstMultiplier:     2.0,
		WindowSize:          10,
		RandomnessFactor:    0.0,
	}

	// Create test dataset
	dataset := &blockchain.DataSet{
		StartBlock:      100,
		EndBlock:        102,
		InitialBaseFee:  1000000000,
		InitialGasLimit: 30000000,
		Blocks: []blockchain.BlockData{
			{
				Number:        100,
				GasLimit:      30000000,
				GasUsed:       15000000,
				BaseFeePerGas: 1000000000,
				Transactions: []blockchain.Transaction{
					{
						Hash:         "0x123",
						Gas:          21000,
						GasUsed:      21000,
						MaxFeePerGas: 2000000000,
						Status:       1,
					},
				},
			},
			{
				Number:        101,
				GasLimit:      30000000,
				GasUsed:       16000000,
				BaseFeePerGas: 1100000000,
				Transactions: []blockchain.Transaction{
					{
						Hash:         "0x456",
						Gas:          50000,
						GasUsed:      45000,
						MaxFeePerGas: 1500000000,
						Status:       1,
					},
				},
			},
			{
				Number:        102,
				GasLimit:      30000000,
				GasUsed:       14000000,
				BaseFeePerGas: 950000000,
				Transactions: []blockchain.Transaction{
					{
						Hash:         "0x789",
						Gas:          30000,
						GasUsed:      28000,
						MaxFeePerGas: 1200000000,
						Status:       1,
					},
				},
			},
		},
	}

	generator := NewGenerator()
	testFile := "test_base_comparison.html"

	// Clean up any existing test files
	defer func() {
		os.Remove(testFile)
		gasFile := strings.Replace(testFile, ".html", "_gas.html", 1)
		os.Remove(gasFile)
	}()

	simulator := blockchain.NewSimulator(cfg)
	simResult, err := simulator.SimulateForVisualization(dataset)
	if err != nil {
		t.Fatalf("SimulateForVisualization failed: %v", err)
	}

	err = generator.GenerateBaseComparisonChart(cfg, dataset, simResult, testFile)
	if err != nil {
		t.Fatalf("GenerateBaseComparisonChart failed: %v", err)
	}

	// Verify main file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("Base comparison chart file was not created")
	}

	// Verify gas usage file was also created
	gasFile := strings.Replace(testFile, ".html", "_gas.html", 1)
	if _, err := os.Stat(gasFile); os.IsNotExist(err) {
		t.Fatal("Gas usage chart file was not created")
	}
}

func TestGenerateChartForScenario(t *testing.T) {
	cfg := config.Config{
		TargetBlockSize:     15000000,
		InitialBaseFee:      1000000000,
		MinBaseFee:          1,
		InitialLearningRate: 0.1,
		MinLearningRate:     0.001,
		MaxLearningRate:     0.5,
		Alpha:               0.01,
		Beta:                0.9,
		Delta:               0.000001,
		Gamma:               0.2,
		BurstMultiplier:     2.0,
		WindowSize:          10,
		RandomnessFactor:    0.0,
	}

	scenario := scenarios.Scenario{
		Name:        "Test Scenario",
		Description: "Test scenario",
		Blocks:      []uint64{15000000, 16000000},
	}

	generator := NewGenerator()

	expectedFile := "chart_test_scenario.html"
	defer os.Remove(expectedFile)
	generator.GenerateChartForScenario(cfg, scenario)

	// Should create file
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Error("Chart file should be created when graphs are enabled")
	}
}

func TestChartDataStructures(t *testing.T) {
	// Test ChartData initialization
	data := ChartData{
		BlockNumbers:  []float64{1, 2, 3},
		BaseFees:      []float64{1.0, 1.1, 1.05},
		LearningRates: []float64{0.1, 0.09, 0.081},
		Utilizations:  []float64{50, 55, 52},
		GasUsages:     []float64{15, 16.5, 15.6},
	}

	if len(data.BlockNumbers) != 3 {
		t.Error("ChartData BlockNumbers not properly initialized")
	}
	if len(data.BaseFees) != 3 {
		t.Error("ChartData BaseFees not properly initialized")
	}

	// Test ComparisonData initialization (now from blockchain package)
	compData := blockchain.ComparisonData{
		BlockNumbers:       []float64{1, 2},
		ActualBaseFees:     []float64{1.0, 1.1},
		AIMDBaseFees:       []float64{1.05, 1.15},
		DroppedPercentages: []float64{2.5, 3.0},
		ActualGasUsages:    []float64{15, 16},
		EffectiveGasUsages: []float64{14.5, 15.5},
		LearningRates:      []float64{0.1, 0.09},
	}

	if len(compData.BlockNumbers) != 2 {
		t.Error("ComparisonData BlockNumbers not properly initialized")
	}
	if len(compData.ActualBaseFees) != 2 {
		t.Error("ComparisonData ActualBaseFees not properly initialized")
	}
}

func TestChartOptions(t *testing.T) {
	options := ChartOptions{
		Width:  1200,
		Height: 800,
		Title:  "Test Chart",
	}

	if options.Width != 1200 {
		t.Error("ChartOptions Width not set correctly")
	}
	if options.Height != 800 {
		t.Error("ChartOptions Height not set correctly")
	}
	if options.Title != "Test Chart" {
		t.Error("ChartOptions Title not set correctly")
	}
}
