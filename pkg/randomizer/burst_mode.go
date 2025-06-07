package randomizer

import "math/rand"

type BurstRandomizer struct {
	rng *rand.Rand

	// Config
	burstProbability float64
	burstDurationMin int
	burstDurationMax int
	burstIntensity   float64

	// State
	inBurstMode     bool
	burstBlocksLeft int
}

func NewBurstRandomizer(seed int64, burstProbability float64, burstDurationMin int, burstDurationMax int, burstIntensity float64) *BurstRandomizer {
	return &BurstRandomizer{
		rng:              rand.New(rand.NewSource(seed)),
		burstProbability: burstProbability,
		burstDurationMin: burstDurationMin,
		burstDurationMax: burstDurationMax,
		burstIntensity:   burstIntensity,
		inBurstMode:      false,
		burstBlocksLeft:  0,
	}
}

func (s *BurstRandomizer) Reset() {
	s.inBurstMode = false
	s.burstBlocksLeft = 0
}

func (s *BurstRandomizer) AddRandomness(gasUsed uint64, maxBlockSize uint64) uint64 {
	if s.burstProbability == 0 {
		return gasUsed
	}

	if s.inBurstMode {
		s.burstBlocksLeft--
		if s.burstBlocksLeft <= 0 {
			s.inBurstMode = false
		}
	} else {
		if s.rng.Float64() < s.burstProbability {
			s.inBurstMode = true
			s.burstBlocksLeft = s.burstDurationMin + s.rng.Intn(s.burstDurationMax-s.burstDurationMin+1)
		}
	}

	if s.inBurstMode {
		result := uint64(float64(gasUsed) * s.burstIntensity)
		if result > maxBlockSize {
			result = maxBlockSize
		}
		return result
	}
	return gasUsed
}
