package visualization

import (
	"github.com/brianbland/feemarketsim/pkg/blockchain"
	"github.com/brianbland/feemarketsim/pkg/config"
	"github.com/brianbland/feemarketsim/pkg/scenarios"
)

// ChartData holds data for creating AIMD charts
type ChartData struct {
	BlockNumbers  []float64
	BaseFees      []float64
	LearningRates []float64
	Utilizations  []float64
	GasUsages     []float64
}

// Note: ComparisonData is now defined in pkg/blockchain/types.go to avoid duplication

// ChartGenerator defines the interface for generating charts
type ChartGenerator interface {
	GenerateAIMDChart(config config.Config, scenario scenarios.Scenario, filename string) error
	GenerateAIMDChartWithLogScale(config config.Config, scenario scenarios.Scenario, filename string) error
	GenerateBaseComparisonChart(config config.Config, dataset *blockchain.DataSet, simResult *blockchain.SimulationResult, filename string) error
	GenerateBaseComparisonChartWithLogScale(config config.Config, dataset *blockchain.DataSet, simResult *blockchain.SimulationResult, filename string) error
	GenerateChartForScenario(config config.Config, scenario scenarios.Scenario)
	GenerateChartForScenarioWithLogScale(config config.Config, scenario scenarios.Scenario)
}

// Generator implements ChartGenerator interface
type Generator struct{}

// NewGenerator creates a new chart generator
func NewGenerator() ChartGenerator {
	return &Generator{}
}

// ChartOptions contains styling and size options for charts
type ChartOptions struct {
	Width  int
	Height int
	Title  string
}
