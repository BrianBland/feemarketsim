package simulator

import (
	"github.com/brianbland/feemarketsim/pkg/config"
)

// ConvertToEIP1559Config converts config.AdjusterConfigs to EIP1559Config
func ConvertToEIP1559Config(cfg *config.Config) *EIP1559Config {
	return &EIP1559Config{
		TargetBlockSize: cfg.TargetBlockSize,
		BurstMultiplier: cfg.BurstMultiplier,
		InitialBaseFee:  cfg.InitialBaseFee,
		MinBaseFee:      cfg.MinBaseFee,
		MaxFeeChange:    cfg.Adjuster.EIP1559.MaxFeeChange,
	}
}

// ConvertToAIMDConfig converts config.AdjusterConfigs to AIMDConfig
func ConvertToAIMDConfig(cfg *config.Config) *AIMDConfig {
	return &AIMDConfig{
		TargetBlockSize:     cfg.TargetBlockSize,
		BurstMultiplier:     cfg.BurstMultiplier,
		InitialBaseFee:      cfg.InitialBaseFee,
		MinBaseFee:          cfg.MinBaseFee,
		WindowSize:          cfg.WindowSize,
		Gamma:               cfg.Adjuster.AIMD.Gamma,
		InitialLearningRate: cfg.Adjuster.AIMD.InitialLearningRate,
		MaxLearningRate:     cfg.Adjuster.AIMD.MaxLearningRate,
		MinLearningRate:     cfg.Adjuster.AIMD.MinLearningRate,
		Alpha:               cfg.Adjuster.AIMD.Alpha,
		Beta:                cfg.Adjuster.AIMD.Beta,
		Delta:               cfg.Adjuster.AIMD.Delta,
	}
}

// ConvertToPIDConfig converts config.AdjusterConfigs to PIDConfig
func ConvertToPIDConfig(cfg *config.Config) *PIDConfig {
	return &PIDConfig{
		TargetBlockSize: cfg.TargetBlockSize,
		BurstMultiplier: cfg.BurstMultiplier,
		InitialBaseFee:  cfg.InitialBaseFee,
		MinBaseFee:      cfg.MinBaseFee,
		Kp:              cfg.Adjuster.PID.Kp,
		Ki:              cfg.Adjuster.PID.Ki,
		Kd:              cfg.Adjuster.PID.Kd,
		MaxIntegral:     cfg.Adjuster.PID.MaxIntegral,
		MinIntegral:     cfg.Adjuster.PID.MinIntegral,
		MaxFeeChange:    cfg.Adjuster.PID.MaxFeeChange,
		WindowSize:      cfg.WindowSize,
	}
}
