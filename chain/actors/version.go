package actors

import (
	"github.com/filecoin-project/go-state-types/network"
)

type Version int

const (
	// Version0 Version = 0
	Version2 Version = 2
)

// Converts a network version into an actors adt version.
func VersionForNetwork(version network.Version) Version {
	return Version2
}
