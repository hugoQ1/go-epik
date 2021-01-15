module github.com/EpiK-Protocol/go-epik/testplans/lotus-soup

go 1.14

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/EpiK-Protocol/go-epik v0.4.2-0.20210112074643-c720307bf800
	github.com/codeskyblue/go-sh v0.0.0-20200712050446-30169cf553fe
	github.com/davecgh/go-spew v1.1.1
	github.com/drand/drand v1.2.1
	github.com/filecoin-project/go-address v0.0.5-0.20201103152444-f2023ef3f5bb
	github.com/filecoin-project/go-fil-markets v1.1.1
	github.com/filecoin-project/go-jsonrpc v0.1.2
	github.com/filecoin-project/go-state-types v0.0.0-20201102161440-c8033295a1fc
	github.com/filecoin-project/go-storedcounter v0.0.0-20200421200003-1c99c62e8a5b
	github.com/filecoin-project/specs-actors/v2 v2.3.3
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.7.4
	github.com/hashicorp/go-multierror v1.1.0
	github.com/influxdata/influxdb v1.8.3 // indirect
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-ipld-format v0.2.0
	github.com/ipfs/go-log/v2 v2.1.2-0.20200626104915-0016c0b4b3e4
	github.com/ipfs/go-merkledag v0.3.2
	github.com/ipfs/go-unixfs v0.2.4
	github.com/ipld/go-car v0.1.1-0.20201119040415-11b6074b6d4d
	github.com/kpacha/opencensus-influxdb v0.0.0-20181102202715-663e2683a27c
	github.com/libp2p/go-libp2p v0.12.0
	github.com/libp2p/go-libp2p-core v0.7.0
	github.com/libp2p/go-libp2p-pubsub-tracer v0.0.0-20200626141350-e730b32bf1e6
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multiaddr-net v0.2.0
	github.com/testground/sdk-go v0.2.6-0.20201016180515-1e40e1b0ec3a
	go.opencensus.io v0.22.5
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
)

// This will work in all build modes: docker:go, exec:go, and local go build.
// On docker:go and exec:go, it maps to /extra/filecoin-ffi, as it's picked up
// as an "extra source" in the manifest.
replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

// replace github.com/filecoin-project/filecoin-ffi => github.com/EpiK-Protocol/go-epik-ffi v0.30.4-0.20210106120422-5c04c857a46c

// replace github.com/EpiK-Protocol/go-epik/testplans/lotus-soup => ./

replace github.com/filecoin-project/specs-actors/v2 => github.com/EpiK-Protocol/go-epik-actors/v2 v2.0.0-20210106111733-7f3bb5bb57b7

replace github.com/filecoin-project/go-fil-markets => github.com/EpiK-Protocol/go-epik-markets v0.5.3-0.20210108072419-877f3f29979a

replace github.com/filecoin-project/specs-storage => github.com/EpiK-Protocol/go-epik-storage v0.1.1-0.20210109141728-73c1715728b4

replace github.com/supranational/blst => ../../extern/blst

replace launchpad.net/gocheck => github.com/go-check/check v0.0.0-20140401040844-163297374fe1

// replace github.com/EpiK-Protocol/go-epik => ../../
