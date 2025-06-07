package simulator

import (
	"testing"

	"github.com/brianbland/feemarketsim/pkg/config"
)

func TestNewFeeAdjuster(t *testing.T) {
	cfg := config.Default()
	aimdConfig := DefaultAIMDConfig()
	adjuster := NewAIMDFeeAdjuster(aimdConfig)

	if adjuster == nil {
		t.Fatal("NewFeeAdjuster returned nil")
	}

	state := adjuster.GetCurrentState()
	if state.BaseFee != cfg.InitialBaseFee {
		t.Errorf("Expected initial base fee %d, got %d", cfg.InitialBaseFee, state.BaseFee)
	}

	if state.LearningRate != aimdConfig.InitialLearningRate {
		t.Errorf("Expected initial learning rate %f, got %f", aimdConfig.InitialLearningRate, state.LearningRate)
	}
}

func TestProcessBlock(t *testing.T) {
	cfg := config.Default()
	aimdConfig := DefaultAIMDConfig()
	adjuster := NewAIMDFeeAdjuster(aimdConfig)

	initialState := adjuster.GetCurrentState()
	initialBaseFee := initialState.BaseFee

	// Process a block with target gas usage
	adjuster.ProcessBlock(cfg.TargetBlockSize)

	// Should have one block now
	blocks := adjuster.GetBlocks()
	if len(blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(blocks))
	}

	// Base fee shouldn't change much for target usage when window isn't full
	state := adjuster.GetCurrentState()
	if state.BaseFee == 0 {
		t.Error("Base fee should not be zero after processing block")
	}

	// Test with full window
	for i := 0; i < aimdConfig.WindowSize; i++ {
		adjuster.ProcessBlock(cfg.TargetBlockSize * 2) // High usage
	}

	finalState := adjuster.GetCurrentState()

	// Base fee should increase with high usage
	if finalState.BaseFee <= initialBaseFee {
		t.Errorf("Expected base fee to increase with high usage, initial: %d, final: %d",
			initialBaseFee, finalState.BaseFee)
	}
}

func TestReset(t *testing.T) {
	cfg := config.Default()
	aimdConfig := DefaultAIMDConfig()
	adjuster := NewAIMDFeeAdjuster(aimdConfig)

	// Process some blocks
	for i := 0; i < 5; i++ {
		adjuster.ProcessBlock(cfg.TargetBlockSize)
	}

	// Verify we have blocks
	if len(adjuster.GetBlocks()) == 0 {
		t.Fatal("Should have blocks before reset")
	}

	// Reset
	adjuster.Reset()

	// Verify reset state
	blocks := adjuster.GetBlocks()
	if len(blocks) != 0 {
		t.Errorf("Expected 0 blocks after reset, got %d", len(blocks))
	}

	state := adjuster.GetCurrentState()
	if state.BaseFee != cfg.InitialBaseFee {
		t.Errorf("Expected base fee reset to %d, got %d", cfg.InitialBaseFee, state.BaseFee)
	}

	if state.LearningRate != aimdConfig.InitialLearningRate {
		t.Errorf("Expected learning rate reset to %f, got %f", aimdConfig.InitialLearningRate, state.LearningRate)
	}
}

// Test different adjuster types to ensure the interface works correctly
func TestAdjusterTypes(t *testing.T) {
	cfg := config.Default()
	factory := NewAdjusterFactory()

	tests := []struct {
		name         string
		adjusterType AdjusterType
	}{
		{"AIMD", AdjusterTypeAIMD},
		{"EIP1559", AdjusterTypeEIP1559},
		{"PID", AdjusterTypePID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adjuster, err := factory.CreateAdjuster(tt.adjusterType, cfg)
			if err != nil {
				t.Fatalf("Failed to create %s adjuster: %v", tt.name, err)
			}

			// Test basic interface methods
			state := adjuster.GetCurrentState()
			if state.BaseFee == 0 {
				t.Errorf("%s adjuster has zero initial base fee", tt.name)
			}

			maxBlockSize := adjuster.GetMaxBlockSize()
			if maxBlockSize == 0 {
				t.Errorf("%s adjuster has zero max block size", tt.name)
			}

			// Test processing a block
			adjuster.ProcessBlock(cfg.TargetBlockSize)
			blocks := adjuster.GetBlocks()
			if len(blocks) != 1 {
				t.Errorf("%s adjuster: expected 1 block after processing, got %d", tt.name, len(blocks))
			}

			// Test reset
			adjuster.Reset()
			blocksAfterReset := adjuster.GetBlocks()
			if len(blocksAfterReset) != 0 {
				t.Errorf("%s adjuster: expected 0 blocks after reset, got %d", tt.name, len(blocksAfterReset))
			}
		})
	}
}

// Test factory with specific configurations
func TestFactoryWithConfigs(t *testing.T) {
	baseConfig := config.Default()
	factory := NewAdjusterFactory()

	// Create adjuster configs
	adjusterConfigs := &baseConfig.Adjuster

	// Set AIMD config
	adjusterConfigs.AIMD.Gamma = 0.25
	adjusterConfigs.AIMD.InitialLearningRate = 0.1
	adjusterConfigs.AIMD.MaxLearningRate = 0.5
	adjusterConfigs.AIMD.MinLearningRate = 0.001
	adjusterConfigs.AIMD.Alpha = 0.005
	adjusterConfigs.AIMD.Beta = 0.95
	adjusterConfigs.AIMD.Delta = 0

	// Set EIP-1559 config
	adjusterConfigs.EIP1559.MaxFeeChange = 0.2

	// Set PID config
	adjusterConfigs.PID.Kp = 0.15
	adjusterConfigs.PID.Ki = 0.02
	adjusterConfigs.PID.Kd = 0.06

	tests := []struct {
		name         string
		adjusterType AdjusterType
	}{
		{"AIMD with configs", AdjusterTypeAIMD},
		{"EIP1559 with configs", AdjusterTypeEIP1559},
		{"PID with configs", AdjusterTypePID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adjuster, err := factory.CreateAdjusterWithConfigs(tt.adjusterType, &baseConfig)
			if err != nil {
				t.Fatalf("Failed to create %s adjuster with configs: %v", tt.name, err)
			}

			// Verify it works
			state := adjuster.GetCurrentState()
			if state.BaseFee == 0 {
				t.Errorf("%s adjuster has zero initial base fee", tt.name)
			}

			// Process a few blocks
			for i := 0; i < 3; i++ {
				adjuster.ProcessBlock(baseConfig.TargetBlockSize)
			}

			blocks := adjuster.GetBlocks()
			if len(blocks) != 3 {
				t.Errorf("%s adjuster: expected 3 blocks, got %d", tt.name, len(blocks))
			}
		})
	}
}
