package randomizer_test

import (
	"testing"

	"github.com/brianbland/feemarketsim/pkg/randomizer"
)

func TestGaussianNoise(t *testing.T) {
	gaussianNoise := randomizer.NewGaussianNoise(12345, 0.1)
	gasUsed := uint64(1000000)
	maxBlockSize := gasUsed * 3 / 2

	for i := 0; i < 1000; i++ {
		randomizedGasUsed := gaussianNoise.AddRandomness(gasUsed, maxBlockSize)
		if randomizedGasUsed > maxBlockSize {
			t.Errorf("Randomized gas used is greater than max block size: %d", randomizedGasUsed)
		}
		if randomizedGasUsed == gasUsed {
			t.Errorf("Randomized gas used is equal to original gas used: %d", randomizedGasUsed)
		}
		gasUsed = randomizedGasUsed
	}
}
