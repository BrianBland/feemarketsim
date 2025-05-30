package simulator

import (
	"testing"

	"github.com/brianbland/feemarketsim/pkg/config"
)

func TestNewFeeAdjuster(t *testing.T) {
	cfg := config.Default()
	adjuster := NewFeeAdjuster(cfg)

	if adjuster == nil {
		t.Fatal("NewFeeAdjuster returned nil")
	}

	state := adjuster.GetCurrentState()
	if state.BaseFee != cfg.InitialBaseFee {
		t.Errorf("Expected initial base fee %d, got %d", cfg.InitialBaseFee, state.BaseFee)
	}

	if state.LearningRate != cfg.InitialLearningRate {
		t.Errorf("Expected initial learning rate %f, got %f", cfg.InitialLearningRate, state.LearningRate)
	}
}

func TestProcessBlock(t *testing.T) {
	cfg := config.Default()
	adjuster := NewFeeAdjuster(cfg)

	// Set deterministic seed for reproducible tests
	adjuster.SetSeed(12345)

	initialBaseFee := adjuster.baseFee

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
	for i := 0; i < cfg.WindowSize; i++ {
		adjuster.ProcessBlock(cfg.TargetBlockSize * 2) // High usage
	}

	finalState := adjuster.GetCurrentState()

	// Base fee should increase with high usage
	if finalState.BaseFee <= initialBaseFee {
		t.Errorf("Expected base fee to increase with high usage, initial: %d, final: %d",
			initialBaseFee, finalState.BaseFee)
	}
}

func TestAddRandomness(t *testing.T) {
	cfg := config.Default()
	cfg.RandomnessFactor = 0.1
	adjuster := NewFeeAdjuster(cfg)
	adjuster.SetSeed(12345) // Deterministic for testing

	gasUsed := uint64(1000000)

	// With randomness, results should vary
	result1 := adjuster.AddRandomness(gasUsed)
	result2 := adjuster.AddRandomness(gasUsed)

	// Results should be different (with high probability)
	if result1 == result2 {
		t.Log("Warning: Randomness produced same result twice (low probability event)")
	}

	// Results should be within reasonable bounds
	maxAllowed := adjuster.GetMaxBlockSize()
	if result1 > maxAllowed || result2 > maxAllowed {
		t.Errorf("Randomness produced result exceeding max block size")
	}

	// Test with zero randomness
	cfg.RandomnessFactor = 0
	adjuster2 := NewFeeAdjuster(cfg)
	result3 := adjuster2.AddRandomness(gasUsed)

	if result3 != gasUsed {
		t.Errorf("Expected no randomness to return original value %d, got %d", gasUsed, result3)
	}
}

func TestReset(t *testing.T) {
	cfg := config.Default()
	adjuster := NewFeeAdjuster(cfg)

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

	if state.LearningRate != cfg.InitialLearningRate {
		t.Errorf("Expected learning rate reset to %f, got %f", cfg.InitialLearningRate, state.LearningRate)
	}
}
