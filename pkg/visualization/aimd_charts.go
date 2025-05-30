package visualization

import (
	"fmt"
	"os"
	"strings"

	"github.com/brianbland/feemarketsim/pkg/config"
	"github.com/brianbland/feemarketsim/pkg/scenarios"
	"github.com/brianbland/feemarketsim/pkg/simulator"
	"github.com/wcharczuk/go-chart/v2"
)

// GenerateAIMDChart creates a comprehensive chart for AIMD simulation results
func (g *Generator) GenerateAIMDChart(cfg config.Config, scenario scenarios.Scenario, filename string) error {
	adjuster := simulator.NewFeeAdjuster(cfg)

	var data ChartData

	// Collect simulation data
	for i, gasUsed := range scenario.Blocks {
		adjuster.ProcessBlock(gasUsed)
		state := adjuster.GetCurrentState()

		data.BlockNumbers = append(data.BlockNumbers, float64(i+1))
		data.BaseFees = append(data.BaseFees, float64(state.BaseFee)/1e9)          // Convert to Gwei
		data.LearningRates = append(data.LearningRates, state.LearningRate*100)    // Convert to percentage
		data.Utilizations = append(data.Utilizations, state.TargetUtilization*100) // Convert to percentage
		data.GasUsages = append(data.GasUsages, float64(gasUsed)/1e6)              // Convert to millions
	}

	// Create the main chart
	graph := chart.Chart{
		Title:  fmt.Sprintf("AIMD Fee Mechanism: %s", scenario.Name),
		Width:  1200,
		Height: 800,
		Background: chart.Style{
			Padding: chart.Box{
				Top:    40,
				Left:   40,
				Right:  40,
				Bottom: 40,
			},
		},
		XAxis: chart.XAxis{
			Name: "Block Number",
		},
		YAxis: chart.YAxis{
			Name: "Base Fee (Gwei)",
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				Name:    "Base Fee (Gwei)",
				XValues: data.BlockNumbers,
				YValues: data.BaseFees,
				Style: chart.Style{
					StrokeColor: chart.ColorRed,
					StrokeWidth: 2,
				},
			},
			chart.ContinuousSeries{
				Name:    "Learning Rate (%)",
				XValues: data.BlockNumbers,
				YValues: data.LearningRates,
				Style: chart.Style{
					StrokeColor:     chart.ColorBlue,
					StrokeWidth:     1,
					StrokeDashArray: []float64{5, 5},
				},
				YAxis: chart.YAxisSecondary,
			},
		},
	}

	// Add secondary Y-axis for learning rate
	graph.YAxisSecondary = chart.YAxis{
		Name: "Learning Rate (%)",
	}

	// Add legend
	graph.Elements = []chart.Renderable{
		chart.LegendThin(&graph),
	}

	// Save the chart
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := graph.Render(chart.PNG, file); err != nil {
		return fmt.Errorf("failed to render chart: %w", err)
	}

	fmt.Printf("Chart saved to %s\n", filename)
	return nil
}

// GenerateChartForScenario creates a chart for a given scenario
func (g *Generator) GenerateChartForScenario(cfg config.Config, scenario scenarios.Scenario) {
	// Create filename based on scenario name
	filename := fmt.Sprintf("chart_%s.png", strings.ToLower(strings.ReplaceAll(scenario.Name, " ", "_")))

	if err := g.GenerateAIMDChart(cfg, scenario, filename); err != nil {
		fmt.Printf("Warning: failed to generate chart for %s: %v\n", scenario.Name, err)
	}
}
