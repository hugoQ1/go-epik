package node

import (
	"context"
	"errors"
	"os"
	"time"

	metricsi "github.com/ipfs/go-metrics-interface"

	"github.com/EpiK-Protocol/go-epik/chain"
	"github.com/EpiK-Protocol/go-epik/chain/exchange"
	"github.com/EpiK-Protocol/go-epik/chain/store"
	"github.com/EpiK-Protocol/go-epik/chain/vm"
	"github.com/EpiK-Protocol/go-epik/chain/wallet"
	"github.com/EpiK-Protocol/go-epik/node/hello"
	"github.com/EpiK-Protocol/go-epik/system"

	"github.com/filecoin-project/go-state-types/abi"
	logging "github.com/ipfs/go-log/v2"
	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-peerstore/pstoremem"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-fil-markets/discovery"
	discoveryimpl "github.com/filecoin-project/go-fil-markets/discovery/impl"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket/impl/storedask"

	storage2 "github.com/filecoin-project/specs-storage/storage"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/chain/beacon"
	"github.com/EpiK-Protocol/go-epik/chain/gen"
	"github.com/EpiK-Protocol/go-epik/chain/gen/slashfilter"
	"github.com/EpiK-Protocol/go-epik/chain/messagepool"
	"github.com/EpiK-Protocol/go-epik/chain/messagesigner"
	"github.com/EpiK-Protocol/go-epik/chain/metrics"
	"github.com/EpiK-Protocol/go-epik/chain/stmgr"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	ledgerwallet "github.com/EpiK-Protocol/go-epik/chain/wallet/ledger"
	"github.com/EpiK-Protocol/go-epik/chain/wallet/remotewallet"
	sectorstorage "github.com/EpiK-Protocol/go-epik/extern/sector-storage"
	"github.com/EpiK-Protocol/go-epik/extern/sector-storage/ffiwrapper"
	"github.com/EpiK-Protocol/go-epik/extern/sector-storage/stores"
	"github.com/EpiK-Protocol/go-epik/extern/sector-storage/storiface"
	sealing "github.com/EpiK-Protocol/go-epik/extern/storage-sealing"
	"github.com/EpiK-Protocol/go-epik/flowchmgr"
	flowsettler "github.com/EpiK-Protocol/go-epik/flowchmgr/settler"
	"github.com/EpiK-Protocol/go-epik/journal"
	"github.com/EpiK-Protocol/go-epik/lib/peermgr"
	_ "github.com/EpiK-Protocol/go-epik/lib/sigs/bls"
	_ "github.com/EpiK-Protocol/go-epik/lib/sigs/secp"
	"github.com/EpiK-Protocol/go-epik/markets/dealfilter"
	"github.com/EpiK-Protocol/go-epik/markets/storageadapter"
	"github.com/EpiK-Protocol/go-epik/miner"
	"github.com/EpiK-Protocol/go-epik/node/config"
	"github.com/EpiK-Protocol/go-epik/node/impl"
	"github.com/EpiK-Protocol/go-epik/node/impl/common"
	"github.com/EpiK-Protocol/go-epik/node/impl/full"
	"github.com/EpiK-Protocol/go-epik/node/modules"
	"github.com/EpiK-Protocol/go-epik/node/modules/dtypes"
	"github.com/EpiK-Protocol/go-epik/node/modules/helpers"
	"github.com/EpiK-Protocol/go-epik/node/modules/lp2p"
	"github.com/EpiK-Protocol/go-epik/node/modules/testing"
	"github.com/EpiK-Protocol/go-epik/node/repo"
	"github.com/EpiK-Protocol/go-epik/paychmgr"
	"github.com/EpiK-Protocol/go-epik/paychmgr/settler"
	"github.com/EpiK-Protocol/go-epik/storage"
	"github.com/EpiK-Protocol/go-epik/storage/sectorblocks"
)

//nolint:deadcode,varcheck
var log = logging.Logger("builder")

// special is a type used to give keys to modules which
//  can't really be identified by the returned type
type special struct{ id int }

//nolint:golint
var (
	DefaultTransportsKey = special{0}  // Libp2p option
	DiscoveryHandlerKey  = special{2}  // Private type
	AddrsFactoryKey      = special{3}  // Libp2p option
	SmuxTransportKey     = special{4}  // Libp2p option
	RelayKey             = special{5}  // Libp2p option
	SecurityKey          = special{6}  // Libp2p option
	BaseRoutingKey       = special{7}  // fx groups + multiret
	NatPortMapKey        = special{8}  // Libp2p option
	ConnectionManagerKey = special{9}  // Libp2p option
	AutoNATSvcKey        = special{10} // Libp2p option
	BandwidthReporterKey = special{11} // Libp2p option
	ConnGaterKey         = special{12} // libp2p option
)

type invoke int

// Invokes are called in the order they are defined.
//nolint:golint
const (
	// InitJournal at position 0 initializes the journal global var as soon as
	// the system starts, so that it's available for all other components.
	InitJournalKey = invoke(iota)

	// System processes.
	InitMemoryWatchdog
	RunSysMetricsKey

	// libp2p
	PstoreAddSelfKeysKey
	StartListeningKey
	BootstrapKey

	// epik
	SetGenesisKey

	RunHelloKey
	RunChainExchangeKey
	RunChainGraphsync
	RunPeerMgrKey

	HandleIncomingBlocksKey
	HandleIncomingMessagesKey
	// HandleMigrateClientFundsKey
	HandlePaymentChannelManagerKey
	HandleFlowChannelManagerKey

	// miner
	GetParamsKey
	// HandleMigrateProviderFundsKey
	HandleDealsKey
	HandleRetrievalKey
	RunSectorServiceKey
	RunMinerMetricsKey

	// daemon
	ExtractApiKey
	HeadMetricsKey
	SettlePaymentChannelsKey
	SettleFlowChannelsKey
	RunPeerTaggerKey
	SetupFallbackBlockstoresKey

	SetApiEndpointKey

	_nInvokes // keep this last
)

type Settings struct {
	// modules is a map of constructors for DI
	//
	// In most cases the index will be a reflect. Type of element returned by
	// the constructor, but for some 'constructors' it's hard to specify what's
	// the return type should be (or the constructor returns fx group)
	modules map[interface{}]fx.Option

	// invokes are separate from modules as they can't be referenced by return
	// type, and must be applied in correct order
	invokes []fx.Option

	nodeType repo.RepoType

	Online bool // Online option applied
	Config bool // Config option applied
	Lite   bool // Start node in "lite" mode
}

// Basic epik-app services
func defaults() []Option {
	return []Option{
		// global system journal.
		Override(new(journal.DisabledEvents), journal.EnvDisabledEvents),
		Override(new(journal.Journal), modules.OpenFilesystemJournal),

		Override(new(system.MemoryConstraints), modules.MemoryConstraints),
		Override(InitMemoryWatchdog, modules.MemoryWatchdog),

		Override(new(helpers.MetricsCtx), func() context.Context {
			return metricsi.CtxScope(context.Background(), "epik")
		}),

		Override(new(dtypes.ShutdownChan), make(chan struct{})),
	}
}

var LibP2P = Options(
	// Host config
	Override(new(dtypes.Bootstrapper), dtypes.Bootstrapper(false)),

	// Host dependencies
	Override(new(peerstore.Peerstore), pstoremem.NewPeerstore),
	Override(PstoreAddSelfKeysKey, lp2p.PstoreAddSelfKeys),
	Override(StartListeningKey, lp2p.StartListening(config.DefaultFullNode().Libp2p.ListenAddresses)),

	// Host settings
	Override(DefaultTransportsKey, lp2p.DefaultTransports),
	Override(AddrsFactoryKey, lp2p.AddrsFactory(nil, nil)),
	Override(SmuxTransportKey, lp2p.SmuxTransport(true)),
	Override(RelayKey, lp2p.NoRelay()),
	Override(SecurityKey, lp2p.Security(true, false)),

	// Host
	Override(new(lp2p.RawHost), lp2p.Host),
	Override(new(host.Host), lp2p.RoutedHost),
	Override(new(lp2p.BaseIpfsRouting), lp2p.DHTRouting(dht.ModeAuto)),

	Override(DiscoveryHandlerKey, lp2p.DiscoveryHandler),

	// Routing
	Override(new(record.Validator), modules.RecordValidator),
	Override(BaseRoutingKey, lp2p.BaseRouting),
	Override(new(routing.Routing), lp2p.Routing),

	// Services
	Override(NatPortMapKey, lp2p.NatPortMap),
	Override(BandwidthReporterKey, lp2p.BandwidthCounter),
	Override(AutoNATSvcKey, lp2p.AutoNATService),

	// Services (pubsub)
	Override(new(*dtypes.ScoreKeeper), lp2p.ScoreKeeper),
	Override(new(*pubsub.PubSub), lp2p.GossipSub),
	Override(new(*config.Pubsub), func(bs dtypes.Bootstrapper) *config.Pubsub {
		return &config.Pubsub{
			Bootstrapper: bool(bs),
		}
	}),

	// Services (connection management)
	Override(ConnectionManagerKey, lp2p.ConnectionManager(50, 200, 20*time.Second, nil)),
	Override(new(*conngater.BasicConnectionGater), lp2p.ConnGater),
	Override(ConnGaterKey, lp2p.ConnGaterOption),
)

func isType(t repo.RepoType) func(s *Settings) bool {
	return func(s *Settings) bool { return s.nodeType == t }
}

func isFullOrLiteNode(s *Settings) bool { return s.nodeType == repo.FullNode }
func isFullNode(s *Settings) bool       { return s.nodeType == repo.FullNode && !s.Lite }
func isLiteNode(s *Settings) bool       { return s.nodeType == repo.FullNode && s.Lite }

// Chain node provides access to the EpiK blockchain, by setting up a full
// validator node, or by delegating some actions to other nodes (lite mode)
var ChainNode = Options(
	// Full node or lite node
	// TODO: Fix offline mode

	// Consensus settings
	Override(new(dtypes.DrandSchedule), modules.BuiltinDrandConfig),
	Override(new(stmgr.UpgradeSchedule), stmgr.DefaultUpgradeSchedule()),
	Override(new(dtypes.NetworkName), modules.NetworkName),
	Override(new(modules.Genesis), modules.ErrorGenesis),
	Override(new(dtypes.AfterGenesisSet), modules.SetGenesis),
	Override(SetGenesisKey, modules.DoSetGenesis),
	Override(new(beacon.Schedule), modules.RandomSchedule),

	// Network bootstrap
	Override(new(dtypes.BootstrapPeers), modules.BuiltinBootstrap),
	Override(new(dtypes.DrandBootstrap), modules.DrandBootstrap),

	// Consensus: crypto dependencies
	Override(new(ffiwrapper.Verifier), ffiwrapper.ProofVerifier),

	// Consensus: VM
	Override(new(vm.SyscallBuilder), vm.Syscalls),

	// Consensus: Chain storage/access
	Override(new(*store.ChainStore), modules.ChainStore),
	Override(new(*stmgr.StateManager), modules.StateManager),
	Override(new(dtypes.ChainBitswap), modules.ChainBitswap),
	Override(new(dtypes.ChainBlockService), modules.ChainBlockService), // todo: unused

	// Consensus: Chain sync

	// We don't want the SyncManagerCtor to be used as an fx constructor, but rather as a value.
	// It will be called implicitly by the Syncer constructor.
	Override(new(chain.SyncManagerCtor), func() chain.SyncManagerCtor { return chain.NewSyncManager }),
	Override(new(*chain.Syncer), modules.NewSyncer),
	Override(new(exchange.Client), exchange.NewClient),

	// Chain networking
	Override(new(*hello.Service), hello.NewHelloService),
	Override(new(exchange.Server), exchange.NewServer),
	Override(new(*peermgr.PeerMgr), peermgr.NewPeerMgr),

	// Chain mining API dependencies
	Override(new(*slashfilter.SlashFilter), modules.NewSlashFilter),

	// Service: Message Pool
	Override(new(dtypes.DefaultMaxFeeFunc), modules.NewDefaultMaxFeeFunc),
	Override(new(*messagepool.MessagePool), modules.MessagePool),
	Override(new(*dtypes.MpoolLocker), new(dtypes.MpoolLocker)),

	// Shared graphsync (markets, serving chain)
	Override(new(dtypes.Graphsync), modules.Graphsync(config.DefaultFullNode().Client.SimultaneousTransfers)),

	// Service: Wallet
	Override(new(*messagesigner.MessageSigner), messagesigner.NewMessageSigner),
	Override(new(*wallet.LocalWallet), wallet.NewWallet),
	Override(new(wallet.Default), From(new(*wallet.LocalWallet))),
	Override(new(api.WalletAPI), From(new(wallet.MultiWallet))),

	// Service: Payment channels
	Override(new(*paychmgr.Store), paychmgr.NewStore),
	Override(new(*paychmgr.Manager), paychmgr.NewManager),
	Override(HandlePaymentChannelManagerKey, paychmgr.HandleManager),
	Override(SettlePaymentChannelsKey, settler.SettlePaymentChannels),

	// Service: Flow channels
	Override(new(*flowchmgr.Store), flowchmgr.NewStore),
	Override(new(*flowchmgr.Manager), flowchmgr.NewManager),
	Override(HandleFlowChannelManagerKey, flowchmgr.HandleManager),
	Override(SettleFlowChannelsKey, flowsettler.SettlePaymentChannels),

	// Markets (common)
	Override(new(*discoveryimpl.Local), modules.NewLocalDiscovery),

	// Markets (retrieval)
	Override(new(discovery.PeerResolver), modules.RetrievalResolver),
	Override(new(retrievalmarket.RetrievalClient), modules.RetrievalClient),
	Override(new(dtypes.ClientDataTransfer), modules.NewClientGraphsyncDataTransfer),

	// Markets (storage)
	// Override(new(*market.FundManager), market.NewFundManager),
	Override(new(dtypes.ClientDatastore), modules.NewClientDatastore),
	Override(new(storagemarket.StorageClient), modules.StorageClient),
	Override(new(storagemarket.StorageClientNode), storageadapter.NewClientNodeAdapter),
	// Override(HandleMigrateClientFundsKey, modules.HandleMigrateClientFunds),

	// Lite node API
	ApplyIf(isLiteNode,
		Override(new(messagesigner.MpoolNonceAPI), From(new(modules.MpoolNonceAPI))),
		Override(new(full.ChainModuleAPI), From(new(api.GatewayAPI))),
		Override(new(full.GasModuleAPI), From(new(api.GatewayAPI))),
		Override(new(full.MpoolModuleAPI), From(new(api.GatewayAPI))),
		Override(new(full.StateModuleAPI), From(new(api.GatewayAPI))),
		Override(new(stmgr.StateManagerAPI), modules.NewRPCStateManager),
	),

	// Full node API / service startup
	ApplyIf(isFullNode,
		Override(new(messagesigner.MpoolNonceAPI), From(new(*messagepool.MessagePool))),
		Override(new(full.ChainModuleAPI), From(new(full.ChainModule))),
		Override(new(full.GasModuleAPI), From(new(full.GasModule))),
		Override(new(full.MpoolModuleAPI), From(new(full.MpoolModule))),
		Override(new(full.StateModuleAPI), From(new(full.StateModule))),
		Override(new(stmgr.StateManagerAPI), From(new(*stmgr.StateManager))),

		Override(RunHelloKey, modules.RunHello),
		Override(RunChainExchangeKey, modules.RunChainExchange),
		Override(RunPeerMgrKey, modules.RunPeerMgr),
		Override(HandleIncomingMessagesKey, modules.HandleIncomingMessages),
		Override(HandleIncomingBlocksKey, modules.HandleIncomingBlocks),
	),

	Override(RunSysMetricsKey, modules.RunChainSysMetrics),
)

var MinerNode = Options(
	// API dependencies
	Override(new(api.Common), From(new(common.CommonAPI))),
	Override(new(sectorstorage.StorageAuth), modules.StorageAuth),

	// Actor config
	Override(new(dtypes.MinerAddress), modules.MinerAddress),
	Override(new(dtypes.MinerID), modules.MinerID),
	Override(new(abi.RegisteredSealProof), modules.SealProofType),
	Override(new(dtypes.NetworkName), modules.StorageNetworkName),

	// Sector storage
	Override(new(*stores.Index), stores.NewIndex),
	Override(new(stores.SectorIndex), From(new(*stores.Index))),
	Override(new(stores.LocalStorage), From(new(repo.LockedRepo))),
	Override(new(*sectorstorage.Manager), modules.SectorStorage),
	Override(new(sectorstorage.SectorManager), From(new(*sectorstorage.Manager))),
	Override(new(storiface.WorkerReturn), From(new(sectorstorage.SectorManager))),

	// Sector storage: Proofs
	Override(new(ffiwrapper.Verifier), ffiwrapper.ProofVerifier),
	Override(new(storage2.Prover), From(new(sectorstorage.SectorManager))),

	// Sealing
	Override(new(sealing.SectorIDCounter), modules.SectorIDCounter),
	Override(GetParamsKey, modules.GetParams),

	// Mining / proving
	Override(new(*slashfilter.SlashFilter), modules.NewSlashFilter),
	Override(new(*storage.Miner), modules.StorageMiner(config.DefaultStorageMiner().Fees)),
	Override(new(*miner.Miner), modules.SetupBlockProducer),
	Override(new(gen.WinningPoStProver), storage.NewWinningPoStProver),

	Override(new(*storage.AddressSelector), modules.AddressSelector(nil)),

	// Markets
	Override(new(dtypes.StagingMultiDstore), modules.StagingMultiDatastore),
	Override(new(dtypes.StagingBlockstore), modules.StagingBlockstore),
	Override(new(dtypes.StagingDAG), modules.StagingDAG),
	Override(new(dtypes.StagingGraphsync), modules.StagingGraphsync),
	Override(new(dtypes.ProviderPieceStore), modules.NewProviderPieceStore),
	Override(new(*sectorblocks.SectorBlocks), sectorblocks.NewSectorBlocks),

	// Markets (retrieval)
	Override(new(retrievalmarket.RetrievalProvider), modules.RetrievalProvider),
	Override(new(dtypes.RetrievalDealFilter), modules.RetrievalDealFilter(nil)),
	Override(HandleRetrievalKey, modules.HandleRetrieval),

	// Markets (storage)
	Override(new(dtypes.ProviderDataTransfer), modules.NewProviderDAGServiceDataTransfer),
	Override(new(*storedask.StoredAsk), modules.NewStorageAsk),
	Override(new(dtypes.StorageDealFilter), modules.BasicDealFilter(nil)),
	Override(new(storagemarket.StorageProvider), modules.StorageProvider),
	Override(new(*storageadapter.DealPublisher), storageadapter.NewDealPublisher(nil, storageadapter.PublishMsgConfig{})),
	Override(new(storagemarket.StorageProviderNode), storageadapter.NewProviderNodeAdapter(nil)),
	// Override(HandleMigrateProviderFundsKey, modules.HandleMigrateProviderFunds),
	Override(HandleDealsKey, modules.HandleDeals),

	// Config (todo: get a real property system)
	Override(new(dtypes.ConsiderOnlineStorageDealsConfigFunc), modules.NewConsiderOnlineStorageDealsConfigFunc),
	Override(new(dtypes.SetConsiderOnlineStorageDealsConfigFunc), modules.NewSetConsideringOnlineStorageDealsFunc),
	Override(new(dtypes.ConsiderOnlineRetrievalDealsConfigFunc), modules.NewConsiderOnlineRetrievalDealsConfigFunc),
	Override(new(dtypes.SetConsiderOnlineRetrievalDealsConfigFunc), modules.NewSetConsiderOnlineRetrievalDealsConfigFunc),
	Override(new(dtypes.StorageDealPieceCidBlocklistConfigFunc), modules.NewStorageDealPieceCidBlocklistConfigFunc),
	Override(new(dtypes.SetStorageDealPieceCidBlocklistConfigFunc), modules.NewSetStorageDealPieceCidBlocklistConfigFunc),
	Override(new(dtypes.ConsiderOfflineStorageDealsConfigFunc), modules.NewConsiderOfflineStorageDealsConfigFunc),
	Override(new(dtypes.SetConsiderOfflineStorageDealsConfigFunc), modules.NewSetConsideringOfflineStorageDealsFunc),
	Override(new(dtypes.ConsiderOfflineRetrievalDealsConfigFunc), modules.NewConsiderOfflineRetrievalDealsConfigFunc),
	Override(new(dtypes.SetConsiderOfflineRetrievalDealsConfigFunc), modules.NewSetConsiderOfflineRetrievalDealsConfigFunc),
	// Override(new(dtypes.ConsiderVerifiedStorageDealsConfigFunc), modules.NewConsiderVerifiedStorageDealsConfigFunc),
	// Override(new(dtypes.SetConsiderVerifiedStorageDealsConfigFunc), modules.NewSetConsideringVerifiedStorageDealsFunc),
	// Override(new(dtypes.ConsiderUnverifiedStorageDealsConfigFunc), modules.NewConsiderUnverifiedStorageDealsConfigFunc),
	// Override(new(dtypes.SetConsiderUnverifiedStorageDealsConfigFunc), modules.NewSetConsideringUnverifiedStorageDealsFunc),
	Override(new(dtypes.SetSealingConfigFunc), modules.NewSetSealConfigFunc),
	Override(new(dtypes.GetSealingConfigFunc), modules.NewGetSealConfigFunc),
	Override(new(dtypes.SetExpectedSealDurationFunc), modules.NewSetExpectedSealDurationFunc),
	Override(new(dtypes.GetExpectedSealDurationFunc), modules.NewGetExpectedSealDurationFunc),

	Override(RunMinerMetricsKey, modules.RunMinerMetrics),
	Override(RunSysMetricsKey, modules.RunMinerSysMetrics),
)

// Online sets up basic libp2p node
func Online() Option {

	return Options(
		// make sure that online is applied before Config.
		// This is important because Config overrides some of Online units
		func(s *Settings) error { s.Online = true; return nil },
		ApplyIf(func(s *Settings) bool { return s.Config },
			Error(errors.New("the Online option must be set before Config option")),
		),

		LibP2P,

		ApplyIf(isFullOrLiteNode, ChainNode),
		ApplyIf(isType(repo.StorageMiner), MinerNode),
	)
}

func StorageMiner(out *api.StorageMiner) Option {
	return Options(
		ApplyIf(func(s *Settings) bool { return s.Config },
			Error(errors.New("the StorageMiner option must be set before Config option")),
		),
		ApplyIf(func(s *Settings) bool { return s.Online },
			Error(errors.New("the StorageMiner option must be set before Online option")),
		),

		func(s *Settings) error {
			s.nodeType = repo.StorageMiner
			return nil
		},

		func(s *Settings) error {
			resAPI := &impl.StorageMinerAPI{}
			s.invokes[ExtractApiKey] = fx.Populate(resAPI)
			*out = resAPI
			return nil
		},
	)
}

// Config sets up constructors based on the provided Config
func ConfigCommon(cfg *config.Common) Option {
	return Options(
		func(s *Settings) error { s.Config = true; return nil },
		Override(new(dtypes.APIEndpoint), func() (dtypes.APIEndpoint, error) {
			return multiaddr.NewMultiaddr(cfg.API.ListenAddress)
		}),
		Override(SetApiEndpointKey, func(lr repo.LockedRepo, e dtypes.APIEndpoint) error {
			return lr.SetAPIEndpoint(e)
		}),
		Override(new(sectorstorage.URLs), func(e dtypes.APIEndpoint) (sectorstorage.URLs, error) {
			ip := cfg.API.RemoteListenAddress

			var urls sectorstorage.URLs
			urls = append(urls, "http://"+ip+"/remote") // TODO: This makes no assumptions, and probably could...
			return urls, nil
		}),
		ApplyIf(func(s *Settings) bool { return s.Online },
			Override(StartListeningKey, lp2p.StartListening(cfg.Libp2p.ListenAddresses)),
			Override(ConnectionManagerKey, lp2p.ConnectionManager(
				cfg.Libp2p.ConnMgrLow,
				cfg.Libp2p.ConnMgrHigh,
				time.Duration(cfg.Libp2p.ConnMgrGrace),
				cfg.Libp2p.ProtectedPeers)),
			Override(new(*pubsub.PubSub), lp2p.GossipSub),
			Override(new(*config.Pubsub), &cfg.Pubsub),

			ApplyIf(func(s *Settings) bool { return len(cfg.Libp2p.BootstrapPeers) > 0 },
				Override(new(dtypes.BootstrapPeers), modules.ConfigBootstrap(cfg.Libp2p.BootstrapPeers)),
			),
		),
		Override(AddrsFactoryKey, lp2p.AddrsFactory(
			cfg.Libp2p.AnnounceAddresses,
			cfg.Libp2p.NoAnnounceAddresses)),
		Override(new(dtypes.MetadataDS), modules.Datastore(cfg.Backup.DisableMetadataLog)),
	)
}

func ConfigFullNode(c interface{}) Option {
	cfg, ok := c.(*config.FullNode)
	if !ok {
		return Error(xerrors.Errorf("invalid config from repo, got: %T", c))
	}

	ipfsMaddr := cfg.Client.IpfsMAddr
	return Options(
		ConfigCommon(&cfg.Common),

		If(cfg.Client.UseIpfs,
			Override(new(dtypes.ClientBlockstore), modules.IpfsClientBlockstore(ipfsMaddr, cfg.Client.IpfsOnlineMode)),
			If(cfg.Client.IpfsUseForRetrieval,
				Override(new(dtypes.ClientRetrievalStoreManager), modules.ClientBlockstoreRetrievalStoreManager),
			),
		),
		Override(new(dtypes.Graphsync), modules.Graphsync(cfg.Client.SimultaneousTransfers)),

		If(cfg.Metrics.HeadNotifs,
			Override(HeadMetricsKey, metrics.SendHeadNotifs(cfg.Metrics.Nickname)),
		),

		If(cfg.Wallet.RemoteBackend != "",
			Override(new(*remotewallet.RemoteWallet), remotewallet.SetupRemoteWallet(cfg.Wallet.RemoteBackend)),
		),
		If(cfg.Wallet.EnableLedger,
			Override(new(*ledgerwallet.LedgerWallet), ledgerwallet.NewWallet),
		),
		If(cfg.Wallet.DisableLocal,
			Unset(new(*wallet.LocalWallet)),
			Override(new(wallet.Default), wallet.NilDefault),
		),
	)
}

func ConfigStorageMiner(c interface{}) Option {
	cfg, ok := c.(*config.StorageMiner)
	if !ok {
		return Error(xerrors.Errorf("invalid config from repo, got: %T", c))
	}

	return Options(
		ConfigCommon(&cfg.Common),

		If(cfg.Dealmaking.Filter != "",
			Override(new(dtypes.StorageDealFilter), modules.BasicDealFilter(dealfilter.CliStorageDealFilter(cfg.Dealmaking.Filter))),
		),

		If(cfg.Dealmaking.RetrievalFilter != "",
			Override(new(dtypes.RetrievalDealFilter), modules.RetrievalDealFilter(dealfilter.CliRetrievalDealFilter(cfg.Dealmaking.RetrievalFilter))),
		),

		Override(new(*storageadapter.DealPublisher), storageadapter.NewDealPublisher(&cfg.Fees, storageadapter.PublishMsgConfig{
			Period:         time.Duration(cfg.Dealmaking.PublishMsgPeriod),
			MaxDealsPerMsg: cfg.Dealmaking.MaxDealsPerPublishMsg,
		})),
		Override(new(storagemarket.StorageProviderNode), storageadapter.NewProviderNodeAdapter(&cfg.Fees)),

		Override(new(sectorstorage.SealerConfig), cfg.Storage),
		Override(new(*storage.AddressSelector), modules.AddressSelector(&cfg.Addresses)),
		Override(new(*storage.Miner), modules.StorageMiner(cfg.Fees)),
	)
}

func Repo(r repo.Repo) Option {
	return func(settings *Settings) error {
		lr, err := r.Lock(settings.nodeType)
		if err != nil {
			return err
		}
		c, err := lr.Config()
		if err != nil {
			return err
		}

		var cfg *config.Chainstore
		switch settings.nodeType {
		case repo.FullNode:
			cfgp, ok := c.(*config.FullNode)
			if !ok {
				return xerrors.Errorf("invalid config from repo, got: %T", c)
			}
			cfg = &cfgp.Chainstore
		default:
			cfg = &config.Chainstore{}
		}

		return Options(
			Override(new(repo.LockedRepo), modules.LockedRepo(lr)), // module handles closing

			Override(new(dtypes.UniversalBlockstore), modules.UniversalBlockstore),

			If(cfg.EnableSplitstore,
				If(cfg.Splitstore.HotStoreType == "badger",
					Override(new(dtypes.HotBlockstore), modules.BadgerHotBlockstore)),
				Override(new(dtypes.ColdBlockstore), modules.BadgerColdBlockstore),
				Override(new(dtypes.SplitBlockstore), modules.SplitBlockstore(cfg)),
				Override(new(dtypes.ChainBlockstore), modules.ChainSplitBlockstore),
				Override(new(dtypes.StateBlockstore), modules.StateSplitBlockstore),
				Override(new(dtypes.BaseBlockstore), From(new(dtypes.SplitBlockstore))),
				Override(new(dtypes.ExposedBlockstore), From(new(dtypes.SplitBlockstore))),
			),
			If(!cfg.EnableSplitstore,
				Override(new(dtypes.ChainBlockstore), modules.ChainFlatBlockstore),
				Override(new(dtypes.StateBlockstore), modules.StateFlatBlockstore),
				Override(new(dtypes.BaseBlockstore), From(new(dtypes.UniversalBlockstore))),
				Override(new(dtypes.ExposedBlockstore), From(new(dtypes.UniversalBlockstore))),
			),

			If(os.Getenv("EPIK_ENABLE_CHAINSTORE_FALLBACK") == "1",
				Override(new(dtypes.ChainBlockstore), modules.FallbackChainBlockstore),
				Override(new(dtypes.StateBlockstore), modules.FallbackStateBlockstore),
				Override(SetupFallbackBlockstoresKey, modules.InitFallbackBlockstores),
			),

			Override(new(dtypes.ClientImportMgr), modules.ClientImportMgr),
			Override(new(dtypes.ClientMultiDstore), modules.ClientMultiDatastore),

			Override(new(dtypes.ClientBlockstore), modules.ClientBlockstore),
			Override(new(dtypes.ClientRetrievalStoreManager), modules.ClientRetrievalStoreManager),
			Override(new(ci.PrivKey), lp2p.PrivKey),
			Override(new(ci.PubKey), ci.PrivKey.GetPublic),
			Override(new(peer.ID), peer.IDFromPublicKey),

			Override(new(types.KeyStore), modules.KeyStore),

			Override(new(*dtypes.APIAlg), modules.APISecret),

			ApplyIf(isType(repo.FullNode), ConfigFullNode(c)),
			ApplyIf(isType(repo.StorageMiner), ConfigStorageMiner(c)),
		)(settings)
	}
}

type FullOption = Option

func Lite(enable bool) FullOption {
	return func(s *Settings) error {
		s.Lite = enable
		return nil
	}
}

func FullAPI(out *api.FullNode, fopts ...FullOption) Option {
	return Options(
		func(s *Settings) error {
			s.nodeType = repo.FullNode
			return nil
		},
		Options(fopts...),
		func(s *Settings) error {
			resAPI := &impl.FullNodeAPI{}
			s.invokes[ExtractApiKey] = fx.Populate(resAPI)
			*out = resAPI
			return nil
		},
	)
}

type StopFunc func(context.Context) error

// New builds and starts new EpiK node
func New(ctx context.Context, opts ...Option) (StopFunc, error) {
	settings := Settings{
		modules: map[interface{}]fx.Option{},
		invokes: make([]fx.Option, _nInvokes),
	}

	// apply module options in the right order
	if err := Options(Options(defaults()...), Options(opts...))(&settings); err != nil {
		return nil, xerrors.Errorf("applying node options failed: %w", err)
	}

	// gather constructors for fx.Options
	ctors := make([]fx.Option, 0, len(settings.modules))
	for _, opt := range settings.modules {
		ctors = append(ctors, opt)
	}

	// fill holes in invokes for use in fx.Options
	for i, opt := range settings.invokes {
		if opt == nil {
			settings.invokes[i] = fx.Options()
		}
	}

	app := fx.New(
		fx.Options(ctors...),
		fx.Options(settings.invokes...),

		fx.NopLogger,
	)

	// TODO: we probably should have a 'firewall' for Closing signal
	//  on this context, and implement closing logic through lifecycles
	//  correctly
	if err := app.Start(ctx); err != nil {
		// comment fx.NopLogger few lines above for easier debugging
		return nil, xerrors.Errorf("starting node: %w", err)
	}

	return app.Stop, nil
}

// In-memory / testing

func Test() Option {
	return Options(
		Unset(RunPeerMgrKey),
		Unset(new(*peermgr.PeerMgr)),
		Override(new(beacon.Schedule), testing.RandomBeacon),
		Override(new(*storageadapter.DealPublisher), storageadapter.NewDealPublisher(nil, storageadapter.PublishMsgConfig{})),
	)
}
