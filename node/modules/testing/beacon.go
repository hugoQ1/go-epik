package testing

import (
	"time"

	"github.com/EpiK-Protocol/go-epik/build"
	"github.com/EpiK-Protocol/go-epik/chain/beacon"
)

func RandomBeacon() (beacon.RandomBeacon, error) {
	return beacon.NewMockBeacon(time.Duration(build.BlockDelaySecs) * time.Second), nil
}
