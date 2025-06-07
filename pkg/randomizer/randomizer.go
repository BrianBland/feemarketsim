package randomizer

type Randomizer interface {
	AddRandomness(gasUsed uint64, maxBlockSize uint64) uint64
}

type CompoundRandomizer struct {
	randomizers []Randomizer
}

func NewCompoundRandomizer(randomizers ...Randomizer) *CompoundRandomizer {
	return &CompoundRandomizer{randomizers: randomizers}
}

func (r *CompoundRandomizer) AddRandomness(gasUsed uint64, maxBlockSize uint64) uint64 {
	for _, randomizer := range r.randomizers {
		gasUsed = randomizer.AddRandomness(gasUsed, maxBlockSize)
	}
	return gasUsed
}
