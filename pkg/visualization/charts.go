package visualization

import (
	"fmt"
	"os"
	"strings"

	"github.com/brianbland/feemarketsim/pkg/config"
	"github.com/brianbland/feemarketsim/pkg/scenarios"
	"github.com/brianbland/feemarketsim/pkg/simulator"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// GenerateChart creates a comprehensive chart for simulation results
func (g *Generator) GenerateChart(cfg config.Config, scenario scenarios.Scenario, filename string) error {
	return g.GenerateChartWithOptions(cfg, scenario, filename, false)
}

// GenerateChartWithLogScale creates a comprehensive chart for simulation results with logarithmic Y-axis
func (g *Generator) GenerateChartWithLogScale(cfg config.Config, scenario scenarios.Scenario, filename string) error {
	return g.GenerateChartWithOptions(cfg, scenario, filename, true)
}

// GenerateChartWithOptions creates a comprehensive chart for simulation results with configurable Y-axis scaling
func (g *Generator) GenerateChartWithOptions(cfg config.Config, scenario scenarios.Scenario, filename string, useLogScale bool) error {
	adjusterType, err := simulator.ParseAdjusterType(cfg.Simulation.AdjusterType)
	if err != nil {
		fmt.Printf("Error: Invalid adjuster type: %v\n", err)
		return err
	}
	adjuster, err := simulator.NewAdjusterFactory().CreateAdjusterWithConfigs(adjusterType, &cfg)
	if err != nil {
		return fmt.Errorf("failed to create adjuster: %w", err)
	}

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

	// Create line chart
	line := charts.NewLine()

	// Determine Y-axis type and additional options
	yAxisType := "value"
	var yAxisOpts opts.YAxis

	if useLogScale {
		yAxisType = "log"
		// When using log scale, we need to handle zero values and set a minimum
		// ECharts log scale doesn't handle zero values well, so we set a small minimum
		yAxisOpts = opts.YAxis{
			Name: "Base Fee (Gwei) - Log Scale",
			Type: yAxisType,
			Min:  1e-6, // Small minimum to avoid log(0) issues
		}
	} else {
		yAxisOpts = opts.YAxis{
			Name: "Base Fee (Gwei)",
			Type: yAxisType,
		}
	}

	// Set global options
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1200px",
			Height: "800px",
		}),
		charts.WithTitleOpts(opts.Title{
			Title: fmt.Sprintf("Simulated Fee Mechanism: %s", scenario.Name),
			Subtitle: func() string {
				if useLogScale {
					return "Base Fee and Learning Rate Analysis - Logarithmic Scale"
				}
				return "Base Fee and Learning Rate Analysis"
			}(),
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Block Number",
			Type: "value",
		}),
		charts.WithYAxisOpts(yAxisOpts),
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

	// Add second Y-axis for learning rate (positioned on the right)
	line.ExtendYAxis(
		opts.YAxis{
			Name:     "Learning Rate (%)",
			Type:     "value",
			Position: "right",
			SplitLine: &opts.SplitLine{
				Show: opts.Bool(false), // Hide grid lines for secondary axis to reduce clutter
			},
		},
	)

	// Prepare data for line series with [x, y] coordinate pairs
	// When using log scale, we need to handle zero/negative values
	baseFeeData := make([]opts.LineData, 0, len(data.BaseFees))
	for i, fee := range data.BaseFees {
		// For log scale, replace zero or negative values with a small positive number
		displayFee := fee
		if useLogScale && fee <= 0 {
			displayFee = 1e-9 // Very small value instead of zero
		}
		baseFeeData = append(baseFeeData, opts.LineData{Value: []interface{}{data.BlockNumbers[i], displayFee}})
	}

	learningRateData := make([]opts.LineData, len(data.LearningRates))
	for i, rate := range data.LearningRates {
		learningRateData[i] = opts.LineData{Value: []interface{}{data.BlockNumbers[i], rate}}
	}

	// Add series with coordinate data - base fees use primary Y-axis (index 0)
	line.AddSeries("Base Fee (Gwei)", baseFeeData,
		charts.WithLineChartOpts(opts.LineChart{
			Smooth: opts.Bool(true),
		}),
	).
		AddSeries("Learning Rate (%)", learningRateData,
			charts.WithLineChartOpts(opts.LineChart{
				YAxisIndex: 1, // Use second Y-axis (right side)
				Smooth:     opts.Bool(true),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Type: "dashed",
			}),
		)

	// Ensure filename has .html extension
	if !strings.HasSuffix(filename, ".html") {
		filename = strings.TrimSuffix(filename, ".png") + ".html"
	}

	// Save the chart
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := line.Render(file); err != nil {
		return fmt.Errorf("failed to render chart: %w", err)
	}

	scaleType := "linear"
	if useLogScale {
		scaleType = "logarithmic"
	}
	fmt.Printf("Interactive chart (%s scale) saved to %s\n", scaleType, filename)
	return nil
}

// GenerateChartForScenario creates a chart for a given scenario
func (g *Generator) GenerateChartForScenario(cfg config.Config, scenario scenarios.Scenario) {
	// Create filename based on scenario name - use .html extension for interactive charts
	filename := fmt.Sprintf("chart_%s.html", strings.ToLower(strings.ReplaceAll(scenario.Name, " ", "_")))

	if err := g.GenerateChart(cfg, scenario, filename); err != nil {
		fmt.Printf("Warning: failed to generate chart for %s: %v\n", scenario.Name, err)
	}
}

// GenerateChartForScenarioWithLogScale creates a chart with log scale for a given scenario
func (g *Generator) GenerateChartForScenarioWithLogScale(cfg config.Config, scenario scenarios.Scenario) {
	// Create filename based on scenario name - use .html extension for interactive charts
	filename := fmt.Sprintf("chart_%s_log.html", strings.ToLower(strings.ReplaceAll(scenario.Name, " ", "_")))

	if err := g.GenerateChartWithLogScale(cfg, scenario, filename); err != nil {
		fmt.Printf("Warning: failed to generate log scale chart for %s: %v\n", scenario.Name, err)
	}
}
