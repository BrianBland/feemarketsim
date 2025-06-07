package randomizer

import (
	"math/rand"
)

// GaussianNoise provides gaussian noise for scenarios
type GaussianNoise struct {
	rng    *rand.Rand
	stdDev float64
}

// NewGaussianNoise creates a new gaussian noise generator
func NewGaussianNoise(seed int64, stdDev float64) *GaussianNoise {
	return &GaussianNoise{
		rng:    rand.New(rand.NewSource(seed)),
		stdDev: stdDev,
	}
}

// AddRandomness adds gaussian noise to a gas usage value
func (s *GaussianNoise) AddRandomness(gasUsed uint64, maxBlockSize uint64) uint64 {
	if s.stdDev == 0 {
		return gasUsed
	}

	// Generate gaussian noise with mean=0, std=stdDev
	noise := s.rng.NormFloat64() * s.stdDev
	multiplier := 1.0 + noise

	// Ensure we don't go below 0 or above burst capacity
	result := uint64(float64(gasUsed) * multiplier)
	if result > maxBlockSize {
		result = maxBlockSize
	}

	return result
}
