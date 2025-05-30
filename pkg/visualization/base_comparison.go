package visualization

import (
	"fmt"
	"os"
	"strings"

	"github.com/brianbland/feemarketsim/pkg/blockchain"
	"github.com/brianbland/feemarketsim/pkg/config"
	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

// GenerateBaseComparisonChart creates a comparison chart between Base and AIMD mechanisms
func (g *Generator) GenerateBaseComparisonChart(cfg config.Config, dataset *blockchain.DataSet, simResult *blockchain.SimulationResult, filename string) error {
	if simResult.ComparisonData == nil {
		return fmt.Errorf("simulation did not collect visualization data")
	}

	data := simResult.ComparisonData

	// Create the comparison plot
	graph := chart.Chart{
		Title:  fmt.Sprintf("Base vs AIMD Fee Comparison (Blocks %d-%d)", dataset.StartBlock, dataset.EndBlock),
		Width:  1400,
		Height: 1000,
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
				Name:    "Actual Base Fees",
				XValues: data.BlockNumbers,
				YValues: data.ActualBaseFees,
				Style: chart.Style{
					StrokeColor: chart.ColorOrange,
					StrokeWidth: 3,
				},
			},
			chart.ContinuousSeries{
				Name:    "AIMD Fees",
				XValues: data.BlockNumbers,
				YValues: data.AIMDBaseFees,
				Style: chart.Style{
					StrokeColor:     drawing.Color{R: 0, G: 150, B: 255, A: 255},
					StrokeWidth:     2,
					StrokeDashArray: []float64{8, 4},
				},
			},
			chart.ContinuousSeries{
				Name:    "Dropped Tx %",
				XValues: data.BlockNumbers,
				YValues: data.DroppedPercentages,
				Style: chart.Style{
					StrokeColor: chart.ColorRed,
					StrokeWidth: 1,
					FillColor:   drawing.Color{R: 255, G: 0, B: 0, A: 50},
				},
				YAxis: chart.YAxisSecondary,
			},
		},
	}

	// Add secondary Y-axis for dropped transactions
	graph.YAxisSecondary = chart.YAxis{
		Name: "Dropped Tx %",
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

	fmt.Printf("Base comparison chart saved to %s\n", filename)

	// Also generate a detailed gas usage comparison
	gasFilename := strings.Replace(filename, ".png", "_gas.png", 1)
	if err := g.generateGasUsageComparison(data, gasFilename, dataset); err != nil {
		fmt.Printf("Warning: failed to generate gas usage chart: %v\n", err)
	}

	return nil
}

// generateGasUsageComparison creates a separate chart for gas usage analysis
func (g *Generator) generateGasUsageComparison(data *blockchain.ComparisonData, filename string, dataset *blockchain.DataSet) error {
	graph := chart.Chart{
		Title:  fmt.Sprintf("Gas Usage Analysis (Blocks %d-%d)", dataset.StartBlock, dataset.EndBlock),
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
			Name: "Gas Usage (Millions)",
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				Name:    "Actual Gas Usage",
				XValues: data.BlockNumbers,
				YValues: data.ActualGasUsages,
				Style: chart.Style{
					StrokeColor: chart.ColorGreen,
					StrokeWidth: 2,
				},
			},
			chart.ContinuousSeries{
				Name:    "Effective Gas (after drops)",
				XValues: data.BlockNumbers,
				YValues: data.EffectiveGasUsages,
				Style: chart.Style{
					StrokeColor:     chart.ColorRed,
					StrokeWidth:     2,
					StrokeDashArray: []float64{5, 5},
				},
			},
		},
	}

	// Add target line if we can estimate it
	if len(data.BlockNumbers) > 0 {
		targetGas := float64(dataset.InitialGasLimit) / 2 / 1e6 // Assume 50% target
		targetValues := make([]float64, len(data.BlockNumbers))
		for i := range targetValues {
			targetValues[i] = targetGas
		}

		targetSeries := chart.ContinuousSeries{
			Name:    "Target Gas Usage",
			XValues: data.BlockNumbers,
			YValues: targetValues,
			Style: chart.Style{
				StrokeColor:     drawing.Color{R: 100, G: 100, B: 100, A: 255},
				StrokeWidth:     1,
				StrokeDashArray: []float64{10, 5},
			},
		}
		graph.Series = append(graph.Series, targetSeries)
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

	return graph.Render(chart.PNG, file)
}
