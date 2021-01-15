package genesis_test

import (
	"testing"

	"github.com/EpiK-Protocol/go-epik/chain/gen/genesis"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/stretchr/testify/assert"
)

func TestGeneratePresealFilePieceCID(t *testing.T) {
	pid, err := genesis.GeneratePaddedPresealFileCID(abi.RegisteredSealProof_StackedDrg8MiBV1_1)
	assert.NoError(t, err)
	assert.True(t, pid.String() == "baga6ea4seaqlirknfys4ycybznn4mrnehv5vdlhgaho6mdjgepo5oxbqczvqmfa")

	pid, err = genesis.GeneratePaddedPresealFileCID(abi.RegisteredSealProof_StackedDrg2KiBV1_1)
	assert.NoError(t, err)
	assert.True(t, pid.String() == "baga6ea4seaqg3ripvqeiq4rtqzpqggenazuoxkwo7gbo7byxu3goip3m5jhbuca")
}
