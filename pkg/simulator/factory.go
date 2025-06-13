package simulator

import (
	"fmt"
	"strings"

	"github.com/brianbland/feemarketsim/pkg/config"
)

// AdjusterType represents the type of fee adjuster
type AdjusterType string

const (
	AdjusterTypeAIMD             AdjusterType = "aimd"
	AdjusterTypeEIP1559          AdjusterType = "eip1559"
	AdjusterTypePID              AdjusterType = "pid"
	AdjusterTypeBatcherSlowPID   AdjusterType = "batcher-slow-pid"
	AdjusterTypeSequencerFastPID AdjusterType = "sequencer-fast-pid"
	AdjusterTypeHierarchicalPID  AdjusterType = "hierarchical-pid"
)

// AdjusterFactory creates fee adjusters based on configuration
type AdjusterFactory struct{}

// NewAdjusterFactory creates a new adjuster factory
func NewAdjusterFactory() *AdjusterFactory {
	return &AdjusterFactory{}
}

// CreateAdjuster creates a fee adjuster based on the specified type and config
func (f *AdjusterFactory) CreateAdjuster(adjusterType AdjusterType, cfg config.Config) (FeeAdjuster, error) {
	switch adjusterType {
	case AdjusterTypeAIMD:
		aimdConfig := &AIMDConfig{
			TargetBlockSize:     cfg.TargetBlockSize,
			BurstMultiplier:     cfg.BurstMultiplier,
			InitialBaseFee:      cfg.InitialBaseFee,
			MinBaseFee:          cfg.MinBaseFee,
			WindowSize:          10,
			InitialLearningRate: 0.1,
			MaxLearningRate:     0.5,
			MinLearningRate:     0.001,
			Alpha:               0.005,
			Beta:                0.95,
			Gamma:               0.25,
			Delta:               0,
		}
		return NewAIMDFeeAdjuster(aimdConfig), nil

	case AdjusterTypeEIP1559:
		eipConfig := &EIP1559Config{
			TargetBlockSize: cfg.TargetBlockSize,
			BurstMultiplier: cfg.BurstMultiplier,
			InitialBaseFee:  cfg.InitialBaseFee,
			MinBaseFee:      cfg.MinBaseFee,
			MaxFeeChange:    0.125,
		}
		return NewEIP1559FeeAdjuster(eipConfig), nil

	case AdjusterTypePID:
		pidConfig := &PIDConfig{
			TargetBlockSize: cfg.TargetBlockSize,
			BurstMultiplier: cfg.BurstMultiplier,
			InitialBaseFee:  cfg.InitialBaseFee,
			MinBaseFee:      cfg.MinBaseFee,
			Kp:              0.1,
			Ki:              0.01,
			Kd:              0.05,
			MaxIntegral:     1000.0,
			MinIntegral:     -1000.0,
			MaxFeeChange:    0.25,
			WindowSize:      3,
		}
		return NewPIDFeeAdjuster(pidConfig), nil

	case AdjusterTypeBatcherSlowPID:
		batcherConfig := DefaultBatcherSlowPIDConfig()
		batcherConfig.TargetBlockSize = cfg.TargetBlockSize
		batcherConfig.BurstMultiplier = cfg.BurstMultiplier
		batcherConfig.InitialBaseFee = cfg.InitialBaseFee
		batcherConfig.MinBaseFee = cfg.MinBaseFee
		return NewBatcherSlowPID(batcherConfig), nil

	case AdjusterTypeSequencerFastPID:
		fastPIDConfig := DefaultSequencerFastPIDConfig()
		fastPIDConfig.TargetBlockSize = cfg.TargetBlockSize
		fastPIDConfig.BurstMultiplier = cfg.BurstMultiplier
		fastPIDConfig.InitialBaseFee = cfg.InitialBaseFee
		fastPIDConfig.MinBaseFee = cfg.MinBaseFee
		return NewSequencerFastPID(fastPIDConfig), nil

	case AdjusterTypeHierarchicalPID:
		hierarchicalConfig := DefaultHierarchicalPIDConfig()
		hierarchicalConfig.TargetBlockSize = cfg.TargetBlockSize
		hierarchicalConfig.BurstMultiplier = cfg.BurstMultiplier
		hierarchicalConfig.InitialBaseFee = cfg.InitialBaseFee
		hierarchicalConfig.MinBaseFee = cfg.MinBaseFee
		return NewHierarchicalPID(hierarchicalConfig), nil

	default:
		return nil, fmt.Errorf("unknown adjuster type: %s", adjusterType)
	}
}

// CreateAdjusterWithConfigs creates a fee adjuster with detailed configuration
func (f *AdjusterFactory) CreateAdjusterWithConfigs(adjusterType AdjusterType, cfg *config.Config) (FeeAdjuster, error) {
	switch adjusterType {
	case AdjusterTypeAIMD:
		aimdConfig := ConvertToAIMDConfig(cfg)
		return NewAIMDFeeAdjuster(aimdConfig), nil

	case AdjusterTypeEIP1559:
		eipConfig := ConvertToEIP1559Config(cfg)
		return NewEIP1559FeeAdjuster(eipConfig), nil

	case AdjusterTypePID:
		pidConfig := ConvertToPIDConfig(cfg)
		return NewPIDFeeAdjuster(pidConfig), nil

	case AdjusterTypeBatcherSlowPID:
		batcherConfig := DefaultBatcherSlowPIDConfig()
		batcherConfig.TargetBlockSize = cfg.TargetBlockSize
		batcherConfig.BurstMultiplier = cfg.BurstMultiplier
		batcherConfig.InitialBaseFee = cfg.InitialBaseFee
		batcherConfig.MinBaseFee = cfg.MinBaseFee
		return NewBatcherSlowPID(batcherConfig), nil

	case AdjusterTypeSequencerFastPID:
		fastPIDConfig := DefaultSequencerFastPIDConfig()
		fastPIDConfig.TargetBlockSize = cfg.TargetBlockSize
		fastPIDConfig.BurstMultiplier = cfg.BurstMultiplier
		fastPIDConfig.InitialBaseFee = cfg.InitialBaseFee
		fastPIDConfig.MinBaseFee = cfg.MinBaseFee
		return NewSequencerFastPID(fastPIDConfig), nil

	case AdjusterTypeHierarchicalPID:
		hierarchicalConfig := DefaultHierarchicalPIDConfig()
		hierarchicalConfig.TargetBlockSize = cfg.TargetBlockSize
		hierarchicalConfig.BurstMultiplier = cfg.BurstMultiplier
		hierarchicalConfig.InitialBaseFee = cfg.InitialBaseFee
		hierarchicalConfig.MinBaseFee = cfg.MinBaseFee
		return NewHierarchicalPID(hierarchicalConfig), nil

	default:
		return nil, fmt.Errorf("unknown adjuster type: %s", adjusterType)
	}
}

// GetAvailableTypes returns a list of available adjuster types
func (f *AdjusterFactory) GetAvailableTypes() []AdjusterType {
	return []AdjusterType{
		AdjusterTypeAIMD,
		AdjusterTypeEIP1559,
		AdjusterTypePID,
		AdjusterTypeBatcherSlowPID,
		AdjusterTypeSequencerFastPID,
		AdjusterTypeHierarchicalPID,
	}
}

// GetTypeDescription returns a description for each adjuster type
func (f *AdjusterFactory) GetTypeDescription(adjusterType AdjusterType) string {
	switch adjusterType {
	case AdjusterTypeAIMD:
		return "AIMD (Additive Increase Multiplicative Decrease) - Original adaptive algorithm"
	case AdjusterTypeEIP1559:
		return "EIP-1559 - Standard Ethereum fee adjustment mechanism"
	case AdjusterTypePID:
		return "PID Controller - Proportional-Integral-Derivative control system"
	case AdjusterTypeBatcherSlowPID:
		return "Batcher Slow PID - Strategic DA cost management with sequencer coordination"
	case AdjusterTypeSequencerFastPID:
		return "Sequencer Fast PID - Fast DA cost management with sequencer coordination"
	case AdjusterTypeHierarchicalPID:
		return "Hierarchical PID - Two-layer control system combining strategic and tactical adjustments"
	default:
		return "Unknown adjuster type"
	}
}

// ParseAdjusterType parses a string into an AdjusterType
func ParseAdjusterType(s string) (AdjusterType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "aimd":
		return AdjusterTypeAIMD, nil
	case "eip1559", "eip-1559":
		return AdjusterTypeEIP1559, nil
	case "pid":
		return AdjusterTypePID, nil
	case "batcher-slow-pid", "batcher_slow_pid":
		return AdjusterTypeBatcherSlowPID, nil
	case "sequencer-fast-pid":
		return AdjusterTypeSequencerFastPID, nil
	case "hierarchical-pid":
		return AdjusterTypeHierarchicalPID, nil
	default:
		return "", fmt.Errorf("unknown adjuster type: %s", s)
	}
}

// ValidateAdjusterType checks if the adjuster type is valid
func ValidateAdjusterType(adjusterType AdjusterType) error {
	factory := NewAdjusterFactory()
	for _, validType := range factory.GetAvailableTypes() {
		if adjusterType == validType {
			return nil
		}
	}
	return fmt.Errorf("invalid adjuster type: %s", adjusterType)
}
