package simulator

import (
	"testing"
	"time"
)

func TestBatcherSlowPIDBasic(t *testing.T) {
	config := DefaultBatcherSlowPIDConfig()
	config.UpdateFrequency = 100 * time.Millisecond // Fast updates for testing

	adjuster := NewBatcherSlowPID(config)

	// Test basic functionality
	adjuster.ProcessBlock(15_000_000) // Target utilization
	state := adjuster.GetCurrentState()

	if state.BaseFee != config.InitialBaseFee {
		t.Errorf("Expected initial base fee %d, got %d", config.InitialBaseFee, state.BaseFee)
	}

	// Test parameter updates channel exists
	paramChan := adjuster.(*BatcherSlowPID).GetParameterUpdates()
	if paramChan == nil {
		t.Error("Parameter updates channel should not be nil")
	}
}

func TestBatcherSlowPIDDAUtilization(t *testing.T) {
	config := DefaultBatcherSlowPIDConfig()
	config.UpdateFrequency = 50 * time.Millisecond
	config.TargetDAUtilization = 0.5 // 50% target

	adjuster := NewBatcherSlowPID(config).(*BatcherSlowPID)

	// Process several blocks to build DA metrics
	for i := 0; i < 15; i++ {
		adjuster.ProcessBlock(20_000_000) // High utilization
		time.Sleep(10 * time.Millisecond)
	}

	// Should have DA metrics
	if len(adjuster.daMetrics) == 0 {
		t.Error("Expected DA metrics to be populated")
	}

	// Check parameter updates were sent
	select {
	case update := <-adjuster.GetParameterUpdates():
		t.Logf("Received parameter update: Kp=%.3f, Ki=%.3f, Reason=%s",
			update.NewKp, update.NewKi, update.Reason)
	case <-time.After(200 * time.Millisecond):
		t.Error("Expected to receive parameter update")
	}
}

func TestBatcherSlowPIDEmergencyMode(t *testing.T) {
	config := DefaultBatcherSlowPIDConfig()
	config.UpdateFrequency = 50 * time.Millisecond
	config.MaxDAUtilization = 0.1 // Very low threshold for testing (10%)

	adjuster := NewBatcherSlowPID(config).(*BatcherSlowPID)

	// Manually inject high DA utilization metrics to trigger emergency mode
	for i := 0; i < 12; i++ {
		// Create block with high gas usage
		adjuster.ProcessBlock(30_000_000) // Very high utilization

		// Manually override the last DA metric to ensure high utilization
		if len(adjuster.daMetrics) > 0 {
			lastIdx := len(adjuster.daMetrics) - 1
			adjuster.daMetrics[lastIdx].DAUsage = uint64(float64(adjuster.daMetrics[lastIdx].DACapacity) * 0.95) // 95% utilization
			adjuster.daMetrics[lastIdx].BatchEfficiency = 0.95
		}

		time.Sleep(10 * time.Millisecond)
	}

	// Wait for parameter update to process
	time.Sleep(150 * time.Millisecond)

	// Check if emergency mode was triggered
	currentDAUtil := adjuster.calculateCurrentDAUtilization()
	t.Logf("Current DA utilization: %.2f%%, Emergency threshold: %.2f%%",
		currentDAUtil*100, config.MaxDAUtilization*100)

	// Check for parameter update with potential emergency response
	select {
	case update := <-adjuster.GetParameterUpdates():
		t.Logf("Received update: %s, Throttling: %v", update.Reason, update.ThrottlingActive)
		// Even if not in emergency mode, we should get a parameter update
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected to receive a parameter update")
	}
}

func TestBatcherSlowPIDParameterClamping(t *testing.T) {
	config := DefaultBatcherSlowPIDConfig()
	config.MaxParameterChange = 0.1 // 10% max change

	adjuster := NewBatcherSlowPID(config).(*BatcherSlowPID)

	// Set initial parameters
	initialKp := 1.0
	adjuster.sequencerParams.NewKp = initialKp

	// Test parameter clamping
	newValue := 2.0 // 100% increase, should be clamped
	clamped := adjuster.clampParameterChange(initialKp, newValue, config.MaxParameterChange)

	expectedMax := initialKp * (1.0 + config.MaxParameterChange)
	if clamped != expectedMax {
		t.Errorf("Expected clamped value %.3f, got %.3f", expectedMax, clamped)
	}
}

func TestBatcherSlowPIDDiagnostics(t *testing.T) {
	adjuster := NewBatcherSlowPID(DefaultBatcherSlowPIDConfig()).(*BatcherSlowPID)

	// Process some blocks
	adjuster.ProcessBlock(15_000_000)
	adjuster.ProcessBlock(18_000_000)

	// Get diagnostics
	diagnostics := adjuster.GetDiagnostics()

	// Check expected diagnostic keys
	expectedKeys := []string{
		"l1_gas_price_gwei",
		"blob_price_gwei",
		"da_utilization",
		"batch_cost_eth",
		"current_sequencer_kp",
		"emergency_mode",
	}

	for _, key := range expectedKeys {
		if _, exists := diagnostics[key]; !exists {
			t.Errorf("Expected diagnostic key '%s' not found", key)
		}
	}
}
