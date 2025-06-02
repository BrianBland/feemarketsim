package scenarios

import (
	"github.com/brianbland/feemarketsim/pkg/config"
	"github.com/brianbland/feemarketsim/pkg/simulator"
)

// Scenario represents a simulation scenario
type Scenario struct {
	Name        string
	Description string
	Blocks      []uint64 // Gas used per block
}

// Generator handles scenario generation
type Generator struct {
	adjuster *simulator.FeeAdjuster
}

// NewGenerator creates a new scenario generator
func NewGenerator(cfg config.Config) *Generator {
	return &Generator{
		adjuster: simulator.NewFeeAdjuster(cfg),
	}
}

// GenerateAll generates all available scenarios
func (g *Generator) GenerateAll(cfg config.Config) map[string]Scenario {
	scenarios := map[string]Scenario{
		"full":   g.generateFullBlocks(cfg),
		"empty":  g.generateEmptyBlocks(cfg),
		"stable": g.generateStableBlocks(cfg),
		"mixed":  g.generateMixedTraffic(cfg),
	}

	// Apply randomness if configured
	if cfg.RandomnessFactor > 0 {
		for key, scenario := range scenarios {
			scenarios[key] = g.applyRandomness(scenario)
		}
	}

	return scenarios
}

// GetByName returns a specific scenario by name
func (g *Generator) GetByName(name string, cfg config.Config) (Scenario, bool) {
	scenarios := g.GenerateAll(cfg)
	scenario, exists := scenarios[name]
	return scenario, exists
}

// generateFullBlocks creates a scenario with full or nearly-full blocks
func (g *Generator) generateFullBlocks(cfg config.Config) Scenario {
	return Scenario{
		Name:        "Full Blocks",
		Description: "Extended sequence of full or nearly-full blocks to test sustained congestion response",
		Blocks: generateExtendedPattern(cfg.TargetBlockSize, []float64{
			1.8, 1.9, 2.0, 1.95, 2.0, 1.85, 1.9, 2.0, 1.9, 1.8, // Initial burst
			2.0, 1.95, 1.85, 1.9, 2.0, 1.95, 1.9, 1.85, 2.0, 1.9, // Sustained high
			1.8, 1.95, 2.0, 1.9, 1.85, 1.95, 2.0, 1.9, 1.8, 1.95, // Continued pressure
			2.0, 1.9, 1.85, 1.95, 2.0, // Final burst
		}),
	}
}

// generateEmptyBlocks creates a scenario with empty or nearly-empty blocks
func (g *Generator) generateEmptyBlocks(cfg config.Config) Scenario {
	return Scenario{
		Name:        "Empty Blocks",
		Description: "Extended sequence of empty or nearly-empty blocks to test sustained low demand response",
		Blocks: generateExtendedPattern(cfg.TargetBlockSize, []float64{
			0.05, 0.03, 0.08, 0.12, 0.06, 0.09, 0.11, 0.07, 0.10, 0.08, // Very low
			0.02, 0.13, 0.06, 0.09, 0.07, 0.04, 0.11, 0.08, 0.05, 0.12, // Continued low
			0.09, 0.06, 0.04, 0.10, 0.07, 0.08, 0.05, 0.11, 0.09, 0.03, // Still low
			0.12, 0.07, 0.06, 0.08, 0.04, // Final low period
		}),
	}
}

// generateStableBlocks creates a scenario with stable block fullness around target
func (g *Generator) generateStableBlocks(cfg config.Config) Scenario {
	return Scenario{
		Name:        "Stable Half Full",
		Description: "Extended variable block fullness averaging around target block size",
		Blocks: generateExtendedPattern(cfg.TargetBlockSize, []float64{
			0.9, 1.1, 1.05, 0.95, 1.0, 1.15, 0.85, 1.08, 0.98, 1.03, // Around target
			0.97, 1.12, 0.92, 1.06, 0.99, 1.01, 0.96, 1.14, 0.88, 1.09, // Continued stability
			1.04, 0.93, 1.07, 0.98, 1.02, 0.95, 1.13, 0.87, 1.05, 1.01, // More variation
			0.94, 1.08, 0.96, 1.04, 0.98, 1.02, 0.97, 1.06, 0.99, 1.03, // Extended stable
		}),
	}
}

// generateMixedTraffic creates a scenario with mixed traffic patterns
func (g *Generator) generateMixedTraffic(cfg config.Config) Scenario {
	return Scenario{
		Name:        "Mixed Traffic Patterns",
		Description: "Extended combination of high congestion, low demand, and stable periods with realistic transitions",
		Blocks: generateExtendedPattern(cfg.TargetBlockSize, []float64{
			// Initial stable period
			1.0, 0.95, 1.05, 0.98, 1.02, 0.97, 1.04, 0.99, 1.01, 0.96,
			0.97, 1.03, 0.98, 1.02, 0.99, 1.01, 0.96, 1.04, 0.97, 1.03,
			0.98, 1.02, 0.99, 1.01, 0.97, 1.03, 0.98, 1.02, 0.99, 1.01,
			// Gradual increase to congestion
			1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.85, 1.9,
			1.95, 2.0, 1.95, 1.9, 1.85, 1.8, 1.85, 1.9, 1.95, 2.0,
			1.95, 1.9, 1.85, 1.8, 1.85, 1.9, 1.95, 2.0, 1.95, 1.9,
			// Sustained high congestion
			1.9, 2.0, 1.95, 1.85, 1.9, 2.0, 1.95, 1.8, 1.9, 2.0,
			1.95, 1.85, 1.9, 2.0, 1.95, 1.8, 1.9, 2.0, 1.95, 1.85,
			1.9, 2.0, 1.95, 1.8, 1.9, 2.0, 1.95, 1.85, 1.9, 2.0,
			// Gradual return toward normal
			1.7, 1.6, 1.5, 1.4, 1.3, 1.2, 1.1, 1.0, 0.9, 0.8,
			0.85, 1.15, 0.9, 1.1, 0.95, 1.05, 0.8, 1.2, 0.75, 1.25,
			0.7, 1.3, 0.85, 1.15, 0.9, 1.1, 0.95, 1.05, 0.8, 1.2,
			// Low demand period
			0.2, 0.1, 0.15, 0.25, 0.18, 0.12, 0.08, 0.22, 0.16, 0.14,
			0.19, 0.11, 0.17, 0.23, 0.15, 0.13, 0.21, 0.09, 0.20, 0.12,
			0.18, 0.14, 0.16, 0.22, 0.10, 0.19, 0.13, 0.17, 0.21, 0.15,
			// Recovery period
			0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.2,
			1.1, 1.0, 0.95, 1.05, 0.98, 1.02, 0.97, 1.03, 0.99, 1.01,
			0.98, 1.02, 0.97, 1.03, 0.99, 1.01, 0.98, 1.02, 0.97, 1.03,
			// Another congestion spike
			1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 2.0, 1.95,
			1.9, 1.85, 1.8, 1.85, 1.9, 1.95, 2.0, 1.95, 1.9, 1.85,
			1.8, 1.85, 1.9, 1.95, 2.0, 1.95, 1.9, 1.85, 1.8, 1.85,
			// Final stabilization
			1.6, 1.5, 1.4, 1.3, 1.2, 1.1, 1.0, 0.95, 1.05, 0.98,
			1.02, 0.97, 1.03, 0.99, 1.01, 0.98, 1.02, 0.97, 1.03, 0.99,
			1.01, 0.98, 1.02, 0.97, 1.03, 0.99, 1.01, 0.98, 1.02, 0.97,
		}),
	}
}

// applyRandomness applies gaussian noise to a scenario
func (g *Generator) applyRandomness(scenario Scenario) Scenario {
	randomizedBlocks := make([]uint64, len(scenario.Blocks))
	for i, gasUsed := range scenario.Blocks {
		randomizedBlocks[i] = g.adjuster.AddRandomness(gasUsed)
	}

	return Scenario{
		Name:        scenario.Name + " (with randomness)",
		Description: scenario.Description + " - includes gaussian noise variations",
		Blocks:      randomizedBlocks,
	}
}

// generateExtendedPattern creates a sequence of gas usage values based on target multipliers
func generateExtendedPattern(targetBlockSize uint64, multipliers []float64) []uint64 {
	blocks := make([]uint64, len(multipliers))
	for i, multiplier := range multipliers {
		blocks[i] = uint64(float64(targetBlockSize) * multiplier)
	}
	return blocks
}

// GetValidScenarioNames returns a list of all valid scenario names
func GetValidScenarioNames() []string {
	return []string{"all", "full", "empty", "stable", "mixed"}
}
