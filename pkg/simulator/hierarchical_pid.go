package simulator

import (
	"fmt"
	"time"
)

// HierarchicalPIDConfig holds configuration for the hierarchical PID system
type HierarchicalPIDConfig struct {
	// Base configuration
	TargetBlockSize uint64
	BurstMultiplier float64
	InitialBaseFee  uint64
	MinBaseFee      uint64

	// Layer configurations
	SlowLayerConfig *BatcherSlowPIDConfig
	FastLayerConfig *SequencerFastPIDConfig

	// Coordination parameters
	EnableCoordination bool          // Whether to enable layer coordination
	UpdateInterval     time.Duration // How often slow layer sends updates to fast layer
}

// DefaultHierarchicalPIDConfig returns optimized defaults for two-layer control
func DefaultHierarchicalPIDConfig() *HierarchicalPIDConfig {
	return &HierarchicalPIDConfig{
		TargetBlockSize: 15_000_000,
		BurstMultiplier: 2.0,
		InitialBaseFee:  1_000_000_000,
		MinBaseFee:      0,

		SlowLayerConfig: DefaultBatcherSlowPIDConfig(),
		FastLayerConfig: DefaultSequencerFastPIDConfig(),

		EnableCoordination: true,
		UpdateInterval:     30 * time.Second, // Slow layer updates fast layer every 30s
	}
}

// Implement AdjusterConfig interface
func (c *HierarchicalPIDConfig) GetTargetBlockSize() uint64  { return c.TargetBlockSize }
func (c *HierarchicalPIDConfig) GetBurstMultiplier() float64 { return c.BurstMultiplier }
func (c *HierarchicalPIDConfig) GetInitialBaseFee() uint64   { return c.InitialBaseFee }
func (c *HierarchicalPIDConfig) GetMinBaseFee() uint64       { return c.MinBaseFee }

// HierarchicalPID implements a two-layer hierarchical PID control system
type HierarchicalPID struct {
	config *HierarchicalPIDConfig

	// Layer implementations
	slowLayer *BatcherSlowPID   // Strategic layer (batcher)
	fastLayer *SequencerFastPID // Tactical layer (sequencer)

	// Coordination state
	lastUpdateTime time.Time

	// Simulation mode flag (true during simulation, false in real deployment)
	simulationMode bool
}

// NewHierarchicalPID creates a new hierarchical PID controller
func NewHierarchicalPID(cfg *HierarchicalPIDConfig) FeeAdjuster {
	// Configure slow layer
	slowConfig := cfg.SlowLayerConfig
	slowConfig.TargetBlockSize = cfg.TargetBlockSize
	slowConfig.BurstMultiplier = cfg.BurstMultiplier
	slowConfig.InitialBaseFee = cfg.InitialBaseFee
	slowConfig.MinBaseFee = cfg.MinBaseFee

	// Configure fast layer
	fastConfig := cfg.FastLayerConfig
	fastConfig.TargetBlockSize = cfg.TargetBlockSize
	fastConfig.BurstMultiplier = cfg.BurstMultiplier
	fastConfig.InitialBaseFee = cfg.InitialBaseFee
	fastConfig.MinBaseFee = cfg.MinBaseFee

	// Create layer instances
	slowLayer := NewBatcherSlowPID(slowConfig).(*BatcherSlowPID)
	fastLayer := NewSequencerFastPID(fastConfig).(*SequencerFastPID)

	return &HierarchicalPID{
		config:         cfg,
		slowLayer:      slowLayer,
		fastLayer:      fastLayer,
		lastUpdateTime: time.Now(),
		simulationMode: true, // Default to simulation mode
	}
}

// GetMaxBlockSize returns max block size
func (hp *HierarchicalPID) GetMaxBlockSize() uint64 {
	return hp.fastLayer.GetMaxBlockSize() // Use fast layer's max size
}

// ProcessBlock processes a block through both layers in coordination
func (hp *HierarchicalPID) ProcessBlock(gasUsed uint64) {
	// Process block in slow layer (strategic decisions)
	hp.slowLayer.ProcessBlock(gasUsed)

	// Check if it's time to send parameter updates to fast layer
	if hp.config.EnableCoordination && time.Since(hp.lastUpdateTime) >= hp.config.UpdateInterval {
		hp.coordinateLayers()
		hp.lastUpdateTime = time.Now()
	}

	// Process block in fast layer (tactical execution)
	hp.fastLayer.ProcessBlock(gasUsed)
}

// coordinateLayers handles coordination between slow and fast layers
func (hp *HierarchicalPID) coordinateLayers() {
	// Get latest parameter updates from slow layer
	select {
	case paramUpdate := <-hp.slowLayer.GetParameterUpdates():
		// Forward the parameter update to fast layer
		hp.fastLayer.SendParameterUpdate(paramUpdate)

		if hp.simulationMode {
			fmt.Printf("Hierarchical PID: Coordinating layers - %s\n", paramUpdate.Reason)
		}
	default:
		// No parameter updates available from slow layer
	}
}

// GetCurrentState returns the state from the fast layer (execution layer)
func (hp *HierarchicalPID) GetCurrentState() State {
	// The fast layer determines the actual fee and utilization
	return hp.fastLayer.GetCurrentState()
}

// GetBlocks returns blocks from the fast layer (execution layer)
func (hp *HierarchicalPID) GetBlocks() []Block {
	return hp.fastLayer.GetBlocks()
}

// Reset resets both layers
func (hp *HierarchicalPID) Reset() {
	hp.slowLayer.Reset()
	hp.fastLayer.Reset()
	hp.lastUpdateTime = time.Now()
}

// SetSimulationMode sets whether the controller is in simulation mode
func (hp *HierarchicalPID) SetSimulationMode(simulation bool) {
	hp.simulationMode = simulation
}

// GetSlowLayerDiagnostics returns diagnostic information from the slow layer
func (hp *HierarchicalPID) GetSlowLayerDiagnostics() map[string]interface{} {
	return hp.slowLayer.GetDiagnostics()
}

// GetFastLayerDiagnostics returns diagnostic information from the fast layer
func (hp *HierarchicalPID) GetFastLayerDiagnostics() map[string]interface{} {
	return hp.fastLayer.GetDiagnostics()
}

// GetDiagnostics returns comprehensive diagnostic information from both layers
func (hp *HierarchicalPID) GetDiagnostics() map[string]interface{} {
	slowDiagnostics := hp.GetSlowLayerDiagnostics()
	fastDiagnostics := hp.GetFastLayerDiagnostics()

	return map[string]interface{}{
		"slow_layer":             slowDiagnostics,
		"fast_layer":             fastDiagnostics,
		"coordination_enabled":   hp.config.EnableCoordination,
		"last_coordination_time": hp.lastUpdateTime,
		"simulation_mode":        hp.simulationMode,
		"update_interval":        hp.config.UpdateInterval,
	}
}
