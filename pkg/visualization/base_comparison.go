package visualization

import (
	"fmt"
	"os"
	"strings"

	"github.com/brianbland/feemarketsim/pkg/blockchain"
	"github.com/brianbland/feemarketsim/pkg/config"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// GenerateBaseComparisonChart creates a comparison chart between Base and AIMD mechanisms
func (g *Generator) GenerateBaseComparisonChart(cfg config.Config, dataset *blockchain.DataSet, simResult *blockchain.SimulationResult, filename string) error {
	if simResult.ComparisonData == nil {
		return fmt.Errorf("simulation did not collect visualization data")
	}

	data := simResult.ComparisonData

	// Create line chart for comparison
	line := charts.NewLine()

	// Set global options
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1400px",
			Height: "1000px",
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    fmt.Sprintf("Base vs AIMD Fee Comparison (Blocks %d-%d)", dataset.StartBlock, dataset.EndBlock),
			Subtitle: "Fee Mechanism Comparison Analysis",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Block Number",
			Type: "value",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Base Fee (Gwei)",
			Type: "value",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
			Top:  "10%",
		}),
		charts.WithToolboxOpts(opts.Toolbox{
			Show: opts.Bool(true),
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  opts.Bool(true),
					Type:  "png",
					Title: "Save as Image",
				},
				DataZoom: &opts.ToolBoxFeatureDataZoom{
					Show:  opts.Bool(true),
					Title: map[string]string{"zoom": "Zoom", "back": "Back"},
				},
			},
		}),
	)

	// Add second Y-axis for dropped transaction percentage (positioned on the right)
	line.ExtendYAxis(
		opts.YAxis{
			Name:     "Dropped Tx %",
			Type:     "value",
			Position: "right",
			SplitLine: &opts.SplitLine{
				Show: opts.Bool(false), // Hide grid lines for secondary axis to reduce clutter
			},
		},
	)

	// Prepare data for line series with [x, y] coordinate pairs
	actualBaseFeeData := make([]opts.LineData, len(data.ActualBaseFees))
	for i, fee := range data.ActualBaseFees {
		actualBaseFeeData[i] = opts.LineData{Value: []interface{}{data.BlockNumbers[i], fee}}
	}

	aimdBaseFeeData := make([]opts.LineData, len(data.AIMDBaseFees))
	for i, fee := range data.AIMDBaseFees {
		aimdBaseFeeData[i] = opts.LineData{Value: []interface{}{data.BlockNumbers[i], fee}}
	}

	droppedPercentageData := make([]opts.LineData, len(data.DroppedPercentages))
	for i, pct := range data.DroppedPercentages {
		droppedPercentageData[i] = opts.LineData{Value: []interface{}{data.BlockNumbers[i], pct}}
	}

	// Add series with coordinate data - fee series use primary Y-axis (index 0)
	line.AddSeries("Actual Base Fees", actualBaseFeeData,
		charts.WithLineChartOpts(opts.LineChart{
			Smooth: opts.Bool(true),
		}),
		charts.WithLineStyleOpts(opts.LineStyle{
			Width: 3,
		}),
	).
		AddSeries("AIMD Fees", aimdBaseFeeData,
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Width: 2,
				Type:  "dashed",
			}),
		).
		AddSeries("Dropped Tx %", droppedPercentageData,
			charts.WithLineChartOpts(opts.LineChart{
				YAxisIndex: 1, // Use second Y-axis (right side)
				Smooth:     opts.Bool(true),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Width: 1,
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Opacity: 0.3,
			}),
		)

	// Save the chart
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := line.Render(file); err != nil {
		return fmt.Errorf("failed to render chart: %w", err)
	}

	fmt.Printf("Base comparison chart saved to %s\n", filename)

	// Also generate a detailed gas usage comparison
	gasFilename := strings.Replace(filename, ".html", "_gas.html", 1)
	if err := g.generateGasUsageComparison(data, gasFilename, dataset); err != nil {
		fmt.Printf("Warning: failed to generate gas usage chart: %v\n", err)
	}

	return nil
}

// generateGasUsageComparison creates a separate chart for gas usage analysis
func (g *Generator) generateGasUsageComparison(data *blockchain.ComparisonData, filename string, dataset *blockchain.DataSet) error {
	// Create line chart for gas usage
	line := charts.NewLine()

	// Set global options
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1200px",
			Height: "800px",
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    fmt.Sprintf("Gas Usage Analysis (Blocks %d-%d)", dataset.StartBlock, dataset.EndBlock),
			Subtitle: "Gas Usage Patterns and Target Analysis",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Block Number",
			Type: "value",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Gas Usage (Millions)",
			Type: "value",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
			Top:  "10%",
		}),
		charts.WithToolboxOpts(opts.Toolbox{
			Show: opts.Bool(true),
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  opts.Bool(true),
					Type:  "png",
					Title: "Save as Image",
				},
				DataZoom: &opts.ToolBoxFeatureDataZoom{
					Show:  opts.Bool(true),
					Title: map[string]string{"zoom": "Zoom", "back": "Back"},
				},
			},
		}),
	)

	// Prepare data for line series with [x, y] coordinate pairs
	actualGasData := make([]opts.LineData, len(data.ActualGasUsages))
	for i, gas := range data.ActualGasUsages {
		actualGasData[i] = opts.LineData{Value: []interface{}{data.BlockNumbers[i], gas}}
	}

	effectiveGasData := make([]opts.LineData, len(data.EffectiveGasUsages))
	for i, gas := range data.EffectiveGasUsages {
		effectiveGasData[i] = opts.LineData{Value: []interface{}{data.BlockNumbers[i], gas}}
	}

	// Add series with coordinate data (no need to set X-axis for numeric data)
	line.AddSeries("Actual Gas Usage", actualGasData,
		charts.WithLineStyleOpts(opts.LineStyle{
			Width: 2,
		}),
	).
		AddSeries("Effective Gas (after drops)", effectiveGasData,
			charts.WithLineStyleOpts(opts.LineStyle{
				Width: 2,
				Type:  "dashed",
			}),
		)

	// Add target line if we can estimate it
	if len(data.BlockNumbers) > 0 {
		targetGas := float64(dataset.InitialGasLimit) / 2 / 1e6 // Assume 50% target
		targetGasData := make([]opts.LineData, len(data.BlockNumbers))
		for i := range targetGasData {
			targetGasData[i] = opts.LineData{Value: []interface{}{data.BlockNumbers[i], targetGas}}
		}

		line.AddSeries("Target Gas Usage", targetGasData,
			charts.WithLineStyleOpts(opts.LineStyle{
				Width: 1,
				Type:  "dotted",
			}),
		)
	}

	// Set series options for better visualization
	line.SetSeriesOptions(
		charts.WithLineChartOpts(opts.LineChart{
			Smooth: opts.Bool(true),
		}),
	)

	// Save the chart
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return line.Render(file)
}
