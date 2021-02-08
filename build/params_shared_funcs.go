package build

import (
	"github.com/filecoin-project/go-address"

	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/EpiK-Protocol/go-epik/node/modules/dtypes"
)

// Core network constants

func BlocksTopic(netName dtypes.NetworkName) string   { return "/epk/blocks/" + string(netName) }
func MessagesTopic(netName dtypes.NetworkName) string { return "/epk/msgs/" + string(netName) }
func DhtProtocolName(netName dtypes.NetworkName) protocol.ID {
	return protocol.ID("/epk/kad/" + string(netName))
}

func SetAddressNetwork(n address.Network) {
	address.CurrentNetwork = n
}

func MustParseAddress(addr string) address.Address {
	ret, err := address.NewFromString(addr)
	if err != nil {
		panic(err)
	}

	return ret
}
