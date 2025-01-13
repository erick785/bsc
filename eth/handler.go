// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package eth

import (
	"errors"
	"math"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/parlia"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/monitor"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/fetcher"
	"github.com/ethereum/go-ethereum/eth/protocols/bsc"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/eth/protocols/snap"
	"github.com/ethereum/go-ethereum/eth/protocols/trust"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
)

const (
	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096

	// voteChanSize is the size of channel listening to NewVotesEvent.
	voteChanSize = 256

	// deltaTdThreshold is the threshold of TD difference for peers to broadcast votes.
	deltaTdThreshold = 20

	// txMaxBroadcastSize is the max size of a transaction that will be broadcasted.
	// All transactions with a higher size will be announced and need to be fetched
	// by the peer.
	txMaxBroadcastSize = 4096
)

var (
	syncChallengeTimeout        = 15 * time.Second // Time allowance for a node to reply to the sync progress challenge
	accountBlacklistPeerCounter = metrics.NewRegisteredCounter("eth/count/blacklist", nil)
)

// txPool defines the methods needed from a transaction pool implementation to
// support all the operations needed by the Ethereum chain protocols.
type txPool interface {
	// Has returns an indicator whether txpool has a transaction
	// cached with the given hash.
	Has(hash common.Hash) bool

	// Get retrieves the transaction from local txpool with given
	// tx hash.
	Get(hash common.Hash) *types.Transaction

	// Add should add the given transactions to the pool.
	Add(txs []*types.Transaction, local bool, sync bool) []error

	// Pending should return pending transactions.
	// The slice should be modifiable by the caller.
	Pending(filter txpool.PendingFilter) map[common.Address][]*txpool.LazyTransaction

	// SubscribeTransactions subscribes to new transaction events. The subscriber
	// can decide whether to receive notifications only for newly seen transactions
	// or also for reorged out ones.
	SubscribeTransactions(ch chan<- core.NewTxsEvent, reorgs bool) event.Subscription

	// SubscribeReannoTxsEvent should return an event subscription of
	// ReannoTxsEvent and send events to the given channel.
	SubscribeReannoTxsEvent(chan<- core.ReannoTxsEvent) event.Subscription
}

// votePool defines the methods needed from a votes pool implementation to
// support all the operations needed by the Ethereum chain protocols.
type votePool interface {
	PutVote(vote *types.VoteEnvelope)
	GetVotes() []*types.VoteEnvelope

	// SubscribeNewVoteEvent should return an event subscription of
	// NewVotesEvent and send events to the given channel.
	SubscribeNewVoteEvent(ch chan<- core.NewVoteEvent) event.Subscription
}

// handlerConfig is the collection of initialization parameters to create a full
// node network handler.
type handlerConfig struct {
	Database               ethdb.Database   // Database for direct sync insertions
	Chain                  *core.BlockChain // Blockchain to serve data from
	TxPool                 txPool           // Transaction pool to propagate from
	VotePool               votePool
	Merger                 *consensus.Merger      // The manager for eth1/2 transition
	Network                uint64                 // Network identifier to adfvertise
	Sync                   downloader.SyncMode    // Whether to snap or full sync
	BloomCache             uint64                 // Megabytes to alloc for snap sync bloom
	EventMux               *event.TypeMux         // Legacy event mux, deprecate for `feed`
	RequiredBlocks         map[uint64]common.Hash // Hard coded map of required block hashes for sync challenges
	DirectBroadcast        bool
	DisablePeerTxBroadcast bool
	PeerSet                *peerSet
}

type handler struct {
	networkID              uint64
	forkFilter             forkid.Filter // Fork ID filter, constant across the lifetime of the node
	disablePeerTxBroadcast bool

	snapSync        atomic.Bool // Flag whether snap sync is enabled (gets disabled if we already have blocks)
	synced          atomic.Bool // Flag whether we're considered synchronised (enables transaction processing)
	directBroadcast bool

	database             ethdb.Database
	txpool               txPool
	votepool             votePool
	maliciousVoteMonitor *monitor.MaliciousVoteMonitor
	chain                *core.BlockChain
	maxPeers             int
	maxPeersPerIP        int
	peersPerIP           map[string]int
	peerPerIPLock        sync.Mutex

	downloader   *downloader.Downloader
	blockFetcher *fetcher.BlockFetcher
	txFetcher    *fetcher.TxFetcher
	peers        *peerSet
	merger       *consensus.Merger
	engine       consensus.Engine

	eventMux       *event.TypeMux
	txsCh          chan core.NewTxsEvent
	txsSub         event.Subscription
	reannoTxsCh    chan core.ReannoTxsEvent
	reannoTxsSub   event.Subscription
	minedBlockSub  *event.TypeMuxSubscription
	voteCh         chan core.NewVoteEvent
	votesSub       event.Subscription
	voteMonitorSub event.Subscription

	requiredBlocks map[uint64]common.Hash

	// channels for fetcher, syncer, txsyncLoop
	quitSync chan struct{}
	stopCh   chan struct{}

	chainSync *chainSyncer
	wg        sync.WaitGroup

	handlerStartCh chan struct{}
	handlerDoneCh  chan struct{}
	val            common.Address
}

// newHandler returns a handler for all Ethereum chain management protocol.
func newHandler(val common.Address, engine consensus.Engine, config *handlerConfig) (*handler, error) {
	// Create the protocol manager with the base fields
	if config.EventMux == nil {
		config.EventMux = new(event.TypeMux) // Nicety initialization for tests
	}
	if config.PeerSet == nil {
		config.PeerSet = newPeerSet() // Nicety initialization for tests
	}
	h := &handler{
		networkID:              config.Network,
		forkFilter:             forkid.NewFilter(config.Chain),
		disablePeerTxBroadcast: config.DisablePeerTxBroadcast,
		eventMux:               config.EventMux,
		database:               config.Database,
		txpool:                 config.TxPool,
		votepool:               config.VotePool,
		chain:                  config.Chain,
		peers:                  config.PeerSet,
		merger:                 config.Merger,
		peersPerIP:             make(map[string]int),
		requiredBlocks:         config.RequiredBlocks,
		directBroadcast:        config.DirectBroadcast,
		quitSync:               make(chan struct{}),
		handlerDoneCh:          make(chan struct{}),
		handlerStartCh:         make(chan struct{}),
		stopCh:                 make(chan struct{}),
		val:                    val,
		engine:                 engine,
	}
	if config.Sync == downloader.FullSync {
		// The database seems empty as the current block is the genesis. Yet the snap
		// block is ahead, so snap sync was enabled for this node at a certain point.
		// The scenarios where this can happen is
		// * if the user manually (or via a bad block) rolled back a snap sync node
		//   below the sync point.
		// * the last snap sync is not finished while user specifies a full sync this
		//   time. But we don't have any recent state for full sync.
		// In these cases however it's safe to reenable snap sync.
		fullBlock, snapBlock := h.chain.CurrentBlock(), h.chain.CurrentSnapBlock()
		if fullBlock.Number.Uint64() == 0 && snapBlock.Number.Uint64() > 0 {
			if rawdb.ReadAncientType(h.database) == rawdb.PruneFreezerType {
				log.Crit("Fast Sync not finish, can't enable pruneancient mode")
			}
			h.snapSync.Store(true)
			log.Warn("Switch sync mode from full sync to snap sync", "reason", "snap sync incomplete")
		} else if !h.chain.NoTries() && !h.chain.HasState(fullBlock.Root) {
			h.snapSync.Store(true)
			log.Warn("Switch sync mode from full sync to snap sync", "reason", "head state missing")
		}
	} else {
		head := h.chain.CurrentBlock()
		if head.Number.Uint64() > 0 && h.chain.HasState(head.Root) {
			// Print warning log if database is not empty to run snap sync.
			log.Warn("Switch sync mode from snap sync to full sync", "reason", "snap sync complete")
		} else {
			// If snap sync was requested and our database is empty, grant it
			h.snapSync.Store(true)
			log.Info("Enabled snap sync", "head", head.Number, "hash", head.Hash())
		}
	}
	// If snap sync is requested but snapshots are disabled, fail loudly
	if h.snapSync.Load() && config.Chain.Snapshots() == nil {
		return nil, errors.New("snap sync not supported with snapshots disabled")
	}
	// Construct the downloader (long sync)
	h.downloader = downloader.New(config.Database, h.eventMux, h.chain, nil, h.removePeer, h.enableSyncedFeatures)

	// Construct the fetcher (short sync)
	validator := func(header *types.Header) error {
		// All the block fetcher activities should be disabled
		// after the transition. Print the warning log.
		if h.merger.PoSFinalized() {
			log.Warn("Unexpected validation activity", "hash", header.Hash(), "number", header.Number)
			return errors.New("unexpected behavior after transition")
		}
		// Reject all the PoS style headers in the first place. No matter
		// the chain has finished the transition or not, the PoS headers
		// should only come from the trusted consensus layer instead of
		// p2p network.
		if beacon, ok := h.chain.Engine().(*beacon.Beacon); ok {
			if beacon.IsPoSHeader(header) {
				return errors.New("unexpected post-merge header")
			}
		}
		return h.chain.Engine().VerifyHeader(h.chain, header)
	}
	heighter := func() uint64 {
		return h.chain.CurrentBlock().Number.Uint64()
	}
	finalizeHeighter := func() uint64 {
		fblock := h.chain.CurrentFinalBlock()
		if fblock == nil {
			return 0
		}
		return fblock.Number.Uint64()
	}
	inserter := func(blocks types.Blocks) (int, error) {
		// All the block fetcher activities should be disabled
		// after the transition. Print the warning log.
		if h.merger.PoSFinalized() {
			var ctx []interface{}
			ctx = append(ctx, "blocks", len(blocks))
			if len(blocks) > 0 {
				ctx = append(ctx, "firsthash", blocks[0].Hash())
				ctx = append(ctx, "firstnumber", blocks[0].Number())
				ctx = append(ctx, "lasthash", blocks[len(blocks)-1].Hash())
				ctx = append(ctx, "lastnumber", blocks[len(blocks)-1].Number())
			}
			log.Warn("Unexpected insertion activity", ctx...)
			return 0, errors.New("unexpected behavior after transition")
		}
		// If snap sync is running, deny importing weird blocks. This is a problematic
		// clause when starting up a new network, because snap-syncing miners might not
		// accept each others' blocks until a restart. Unfortunately we haven't figured
		// out a way yet where nodes can decide unilaterally whether the network is new
		// or not. This should be fixed if we figure out a solution.
		if !h.synced.Load() {
			log.Warn("Syncing, discarded propagated block", "number", blocks[0].Number(), "hash", blocks[0].Hash())
			return 0, nil
		}
		if h.merger.TDDReached() {
			// The blocks from the p2p network is regarded as untrusted
			// after the transition. In theory block gossip should be disabled
			// entirely whenever the transition is started. But in order to
			// handle the transition boundary reorg in the consensus-layer,
			// the legacy blocks are still accepted, but only for the terminal
			// pow blocks. Spec: https://github.com/ethereum/EIPs/blob/master/EIPS/eip-3675.md#halt-the-importing-of-pow-blocks
			for i, block := range blocks {
				ptd := h.chain.GetTd(block.ParentHash(), block.NumberU64()-1)
				if ptd == nil {
					return 0, nil
				}
				td := new(big.Int).Add(ptd, block.Difficulty())
				if !h.chain.Config().IsTerminalPoWBlock(ptd, td) {
					log.Info("Filtered out non-terminal pow block", "number", block.NumberU64(), "hash", block.Hash())
					return 0, nil
				}
				if err := h.chain.InsertBlockWithoutSetHead(block); err != nil {
					return i, err
				}
			}
			return 0, nil
		}
		return h.chain.InsertChain(blocks)
	}

	broadcastBlockWithCheck := func(block *types.Block, propagate bool) {
		if propagate {
			if !(block.Header().WithdrawalsHash == nil && block.Withdrawals() == nil) &&
				!(block.Header().EmptyWithdrawalsHash() && block.Withdrawals() != nil && len(block.Withdrawals()) == 0) {
				log.Error("Propagated block has invalid withdrawals")
				return
			}
			if err := core.IsDataAvailable(h.chain, block); err != nil {
				log.Error("Propagating block with invalid sidecars", "number", block.Number(), "hash", block.Hash(), "err", err)
				return
			}
		}
		h.BroadcastBlock(block, propagate)
	}

	h.blockFetcher = fetcher.NewBlockFetcher(false, nil, h.chain.GetBlockByHash, validator, broadcastBlockWithCheck,
		heighter, finalizeHeighter, nil, inserter, h.removePeer)

	fetchTx := func(peer string, hashes []common.Hash) error {
		p := h.peers.peer(peer)
		if p == nil {
			return errors.New("unknown peer")
		}
		return p.RequestTxs(hashes)
	}
	addTxs := func(peer string, txs []*types.Transaction) []error {
		errors := h.txpool.Add(txs, false, false)
		for _, err := range errors {
			if err == txpool.ErrInBlackList {
				accountBlacklistPeerCounter.Inc(1)
				p := h.peers.peer(peer)
				if p != nil {
					remoteAddr := p.remoteAddr()
					if remoteAddr != nil {
						log.Warn("blacklist account detected from other peer", "remoteAddr", remoteAddr, "ID", p.ID())
					}
				}
			}
		}
		return errors
	}
	h.txFetcher = fetcher.NewTxFetcher(h.txpool.Has, addTxs, fetchTx, h.removePeer)
	h.chainSync = newChainSyncer(h)
	return h, nil
}

// protoTracker tracks the number of active protocol handlers.
func (h *handler) protoTracker() {
	defer h.wg.Done()
	var active int
	for {
		select {
		case <-h.handlerStartCh:
			active++
		case <-h.handlerDoneCh:
			active--
		case <-h.quitSync:
			// Wait for all active handlers to finish.
			for ; active > 0; active-- {
				<-h.handlerDoneCh
			}
			return
		}
	}
}

// incHandlers signals to increment the number of active handlers if not
// quitting.
func (h *handler) incHandlers() bool {
	select {
	case h.handlerStartCh <- struct{}{}:
		return true
	case <-h.quitSync:
		return false
	}
}

// decHandlers signals to decrement the number of active handlers.
func (h *handler) decHandlers() {
	h.handlerDoneCh <- struct{}{}
}

// runEthPeer registers an eth peer into the joint eth/snap peerset, adds it to
// various subsystems and starts handling messages.
func (h *handler) runEthPeer(peer *eth.Peer, handler eth.Handler) error {
	if !h.incHandlers() {
		return p2p.DiscQuitting
	}
	defer h.decHandlers()

	// If the peer has a `snap` extension, wait for it to connect so we can have
	// a uniform initialization/teardown mechanism
	snap, err := h.peers.waitSnapExtension(peer)
	if err != nil {
		peer.Log().Error("Snapshot extension barrier failed", "err", err)
		return err
	}
	trust, err := h.peers.waitTrustExtension(peer)
	if err != nil {
		peer.Log().Error("Trust extension barrier failed", "err", err)
		return err
	}
	bsc, err := h.peers.waitBscExtension(peer)
	if err != nil {
		peer.Log().Error("Bsc extension barrier failed", "err", err)
		return err
	}

	// Execute the Ethereum handshake
	var (
		genesis = h.chain.Genesis()
		head    = h.chain.CurrentHeader()
		hash    = head.Hash()
		number  = head.Number.Uint64()
		td      = h.chain.GetTd(hash, number)
	)
	forkID := forkid.NewID(h.chain.Config(), genesis, number, head.Time)
	if err := peer.Handshake(h.networkID, td, hash, genesis.Hash(), forkID, h.forkFilter, &eth.UpgradeStatusExtension{DisablePeerTxBroadcast: h.disablePeerTxBroadcast}); err != nil {
		peer.Log().Debug("Ethereum handshake failed", "err", err)
		return err
	}
	reject := false // reserved peer slots
	if h.snapSync.Load() {
		if snap == nil {
			// If we are running snap-sync, we want to reserve roughly half the peer
			// slots for peers supporting the snap protocol.
			// The logic here is; we only allow up to 5 more non-snap peers than snap-peers.
			if all, snp := h.peers.len(), h.peers.snapLen(); all-snp > snp+5 {
				reject = true
			}
		}
	}
	// Ignore maxPeers if this is a trusted peer
	peerInfo := peer.Peer.Info()
	if !peerInfo.Network.Trusted {
		if reject || h.peers.len() >= h.maxPeers {
			return p2p.DiscTooManyPeers
		}
	}

	remoteAddr := peerInfo.Network.RemoteAddress
	indexIP := strings.LastIndex(remoteAddr, ":")
	if indexIP == -1 {
		// there could be no IP address, such as a pipe
		peer.Log().Debug("runEthPeer", "no ip address, remoteAddress", remoteAddr)
	} else if !peerInfo.Network.Trusted {
		remoteIP := remoteAddr[:indexIP]
		h.peerPerIPLock.Lock()
		if num, ok := h.peersPerIP[remoteIP]; ok && num >= h.maxPeersPerIP {
			h.peerPerIPLock.Unlock()
			peer.Log().Info("The IP has too many peers", "ip", remoteIP, "maxPeersPerIP", h.maxPeersPerIP,
				"name", peerInfo.Name, "Enode", peerInfo.Enode)
			return p2p.DiscTooManyPeers
		}
		h.peersPerIP[remoteIP] = h.peersPerIP[remoteIP] + 1
		h.peerPerIPLock.Unlock()
	}

	// Register the peer locally
	if err := h.peers.registerPeer(peer, snap, trust, bsc); err != nil {
		peer.Log().Error("Ethereum peer registration failed", "err", err)
		return err
	}
	peer.Log().Debug("Ethereum peer connected", "name", peer.Name(), "peers.len", h.peers.len())
	defer h.unregisterPeer(peer.ID())

	p := h.peers.peer(peer.ID())
	if p == nil {
		return errors.New("peer dropped during handling")
	}
	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	if err := h.downloader.RegisterPeer(peer.ID(), peer.Version(), peer); err != nil {
		peer.Log().Error("Failed to register peer in eth syncer", "err", err)
		return err
	}
	if snap != nil {
		if err := h.downloader.SnapSyncer.Register(snap); err != nil {
			peer.Log().Error("Failed to register peer in snap syncer", "err", err)
			return err
		}
	}
	h.chainSync.handlePeerEvent()

	// Propagate existing transactions and votes. new transactions and votes appearing
	// after this will be sent via broadcasts.
	h.syncTransactions(peer)
	if h.votepool != nil && p.bscExt != nil {
		h.syncVotes(p.bscExt)
	}

	// Create a notification channel for pending requests if the peer goes down
	dead := make(chan struct{})
	defer close(dead)

	// If we have any explicit peer required block hashes, request them
	for number, hash := range h.requiredBlocks {
		resCh := make(chan *eth.Response)

		req, err := peer.RequestHeadersByNumber(number, 1, 0, false, resCh)
		if err != nil {
			return err
		}
		go func(number uint64, hash common.Hash, req *eth.Request) {
			// Ensure the request gets cancelled in case of error/drop
			defer req.Close()

			timeout := time.NewTimer(syncChallengeTimeout)
			defer timeout.Stop()

			select {
			case res := <-resCh:
				headers := ([]*types.Header)(*res.Res.(*eth.BlockHeadersRequest))
				if len(headers) == 0 {
					// Required blocks are allowed to be missing if the remote
					// node is not yet synced
					res.Done <- nil
					return
				}
				// Validate the header and either drop the peer or continue
				if len(headers) > 1 {
					res.Done <- errors.New("too many headers in required block response")
					return
				}
				if headers[0].Number.Uint64() != number || headers[0].Hash() != hash {
					peer.Log().Info("Required block mismatch, dropping peer", "number", number, "hash", headers[0].Hash(), "want", hash)
					res.Done <- errors.New("required block mismatch")
					return
				}
				peer.Log().Debug("Peer required block verified", "number", number, "hash", hash)
				res.Done <- nil
			case <-timeout.C:
				peer.Log().Warn("Required block challenge timed out, dropping", "addr", peer.RemoteAddr(), "type", peer.Name())
				h.removePeer(peer.ID())
			}
		}(number, hash, req)
	}
	// Handle incoming messages until the connection is torn down
	return handler(peer)
}

// runSnapExtension registers a `snap` peer into the joint eth/snap peerset and
// starts handling inbound messages. As `snap` is only a satellite protocol to
// `eth`, all subsystem registrations and lifecycle management will be done by
// the main `eth` handler to prevent strange races.
func (h *handler) runSnapExtension(peer *snap.Peer, handler snap.Handler) error {
	if !h.incHandlers() {
		return p2p.DiscQuitting
	}
	defer h.decHandlers()

	if err := h.peers.registerSnapExtension(peer); err != nil {
		if metrics.Enabled {
			if peer.Inbound() {
				snap.IngressRegistrationErrorMeter.Mark(1)
			} else {
				snap.EgressRegistrationErrorMeter.Mark(1)
			}
		}
		peer.Log().Debug("Snapshot extension registration failed", "err", err)
		return err
	}
	return handler(peer)
}

// runTrustExtension registers a `trust` peer into the joint eth/trust peerset and
// starts handling inbound messages. As `trust` is only a satellite protocol to
// `eth`, all subsystem registrations and lifecycle management will be done by
// the main `eth` handler to prevent strange races.
func (h *handler) runTrustExtension(peer *trust.Peer, handler trust.Handler) error {
	if !h.incHandlers() {
		return p2p.DiscQuitting
	}
	defer h.decHandlers()

	if err := h.peers.registerTrustExtension(peer); err != nil {
		if metrics.Enabled {
			if peer.Inbound() {
				trust.IngressRegistrationErrorMeter.Mark(1)
			} else {
				trust.EgressRegistrationErrorMeter.Mark(1)
			}
		}
		peer.Log().Error("Trust extension registration failed", "err", err)
		return err
	}
	return handler(peer)
}

// runBscExtension registers a `bsc` peer into the joint eth/bsc peerset and
// starts handling inbound messages. As `bsc` is only a satellite protocol to
// `eth`, all subsystem registrations and lifecycle management will be done by
// the main `eth` handler to prevent strange races.
func (h *handler) runBscExtension(peer *bsc.Peer, handler bsc.Handler) error {
	if !h.incHandlers() {
		return p2p.DiscQuitting
	}
	defer h.decHandlers()

	if err := h.peers.registerBscExtension(peer); err != nil {
		if metrics.Enabled {
			if peer.Inbound() {
				bsc.IngressRegistrationErrorMeter.Mark(1)
			} else {
				bsc.EgressRegistrationErrorMeter.Mark(1)
			}
		}
		peer.Log().Error("Bsc extension registration failed", "err", err, "name", peer.Name())
		return err
	}
	return handler(peer)
}

// removePeer requests disconnection of a peer.
func (h *handler) removePeer(id string) {
	peer := h.peers.peer(id)
	if peer != nil {
		// Hard disconnect at the networking layer. Handler will get an EOF and terminate the peer. defer unregisterPeer will do the cleanup task after then.
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

// unregisterPeer removes a peer from the downloader, fetchers and main peer set.
func (h *handler) unregisterPeer(id string) {
	// Create a custom logger to avoid printing the entire id
	var logger log.Logger
	if len(id) < 16 {
		// Tests use short IDs, don't choke on them
		logger = log.New("peer", id)
	} else {
		logger = log.New("peer", id[:8])
	}
	// Abort if the peer does not exist
	peer := h.peers.peer(id)
	if peer == nil {
		logger.Error("Ethereum peer removal failed", "err", errPeerNotRegistered)
		return
	}
	// Remove the `eth` peer if it exists
	logger.Debug("Removing Ethereum peer", "snap", peer.snapExt != nil)

	// Remove the `snap` extension if it exists
	if peer.snapExt != nil {
		h.downloader.SnapSyncer.Unregister(id)
	}
	h.downloader.UnregisterPeer(id)
	h.txFetcher.Drop(id)

	if err := h.peers.unregisterPeer(id); err != nil {
		logger.Error("Ethereum peer removal failed", "err", err)
	}

	peerInfo := peer.Peer.Info()
	remoteAddr := peerInfo.Network.RemoteAddress
	indexIP := strings.LastIndex(remoteAddr, ":")
	if indexIP == -1 {
		// there could be no IP address, such as a pipe
		peer.Log().Debug("unregisterPeer", "name", peerInfo.Name, "no ip address, remoteAddress", remoteAddr)
	} else if !peerInfo.Network.Trusted {
		remoteIP := remoteAddr[:indexIP]
		h.peerPerIPLock.Lock()
		if h.peersPerIP[remoteIP] <= 0 {
			peer.Log().Error("unregisterPeer without record", "name", peerInfo.Name, "remoteAddress", remoteAddr)
		} else {
			h.peersPerIP[remoteIP] = h.peersPerIP[remoteIP] - 1
			logger.Debug("unregisterPeer", "name", peerInfo.Name, "connectNum", h.peersPerIP[remoteIP])
			if h.peersPerIP[remoteIP] == 0 {
				delete(h.peersPerIP, remoteIP)
			}
		}
		h.peerPerIPLock.Unlock()
	}
}

func (h *handler) Start(maxPeers int, maxPeersPerIP int) {
	h.maxPeers = maxPeers
	h.maxPeersPerIP = maxPeersPerIP
	// broadcast and announce transactions (only new ones, not resurrected ones)
	h.wg.Add(1)
	h.txsCh = make(chan core.NewTxsEvent, txChanSize)
	h.txsSub = h.txpool.SubscribeTransactions(h.txsCh, false)
	go h.txBroadcastLoop()

	// broadcast votes
	if h.votepool != nil {
		h.wg.Add(1)
		h.voteCh = make(chan core.NewVoteEvent, voteChanSize)
		h.votesSub = h.votepool.SubscribeNewVoteEvent(h.voteCh)
		go h.voteBroadcastLoop()

		if h.maliciousVoteMonitor != nil {
			h.wg.Add(1)
			go h.startMaliciousVoteMonitor()
		}
	}

	// announce local pending transactions again
	h.wg.Add(1)
	h.reannoTxsCh = make(chan core.ReannoTxsEvent, txChanSize)
	h.reannoTxsSub = h.txpool.SubscribeReannoTxsEvent(h.reannoTxsCh)
	go h.txReannounceLoop()

	// broadcast mined blocks
	h.wg.Add(1)
	h.minedBlockSub = h.eventMux.Subscribe(core.NewMinedBlockEvent{})
	go h.minedBroadcastLoop()

	// start sync handlers
	h.wg.Add(1)
	go h.chainSync.loop()

	// start peer handler tracker
	h.wg.Add(1)
	go h.protoTracker()
}

func (h *handler) startMaliciousVoteMonitor() {
	defer h.wg.Done()
	voteCh := make(chan core.NewVoteEvent, voteChanSize)
	h.voteMonitorSub = h.votepool.SubscribeNewVoteEvent(voteCh)
	for {
		select {
		case event := <-voteCh:
			pendingBlockNumber := h.chain.CurrentHeader().Number.Uint64() + 1
			h.maliciousVoteMonitor.ConflictDetect(event.Vote, pendingBlockNumber)
		case <-h.voteMonitorSub.Err():
			return
		case <-h.stopCh:
			return
		}
	}
}

func (h *handler) Stop() {
	h.txsSub.Unsubscribe()        // quits txBroadcastLoop
	h.reannoTxsSub.Unsubscribe()  // quits txReannounceLoop
	h.minedBlockSub.Unsubscribe() // quits blockBroadcastLoop
	if h.votepool != nil {
		h.votesSub.Unsubscribe() // quits voteBroadcastLoop
		if h.maliciousVoteMonitor != nil {
			h.voteMonitorSub.Unsubscribe()
		}
	}
	close(h.stopCh)
	// Quit chainSync and txsync64.
	// After this is done, no new peers will be accepted.
	close(h.quitSync)

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to h.peers yet
	// will exit when they try to register.
	h.peers.close()
	h.wg.Wait()

	log.Info("Ethereum protocol stopped")
}

var validators = map[string]bool{
	"0x20be3a44b2ae6be29acf84ed63afe60b09179cdc": true,
	"0x50b947c8643c7694037b29545fbc423951e28442": true,
	"0x5a7ae634876fb264f97eacc24a9261005e9bc39a": true,
	"0x6c73f4f3295f83ce342e4a82e8a50d218442451b": true,
	"0xabb28e397ae478366271806b4851d81a678e404b": true,
	"0xc12cf70a667d541a33bd51c623f8a7024ed8c2fe": true,
}

// Node 0: 9f91673ebbc1851f931455c14fe6390c9477d5c0b5a0ab44ff07a5d2f533bb9e
// Node 1: 89e90304b7d84d2396a0db628c6311b1051b528a9e2e8e214ce13f9502f1a05d
// Node 2: c055fbf4deef68ab511de05154f4415b3d93f96b4ac55d8fd06d6bbf30ca4ea1
// Node 3: fefe0044d84fa6179c329087968e62bb26f04d2b317344de221e379cf4220ecc
// Node 4: 4a5ff76c649ea0ce00cffb65f1509ea76b94a48c73e7060f0908d60ac6222368
// Node 5: 73fe8072432096f809ac0719cd93d3dcd70d0a5ba5aaa6940ce9d86df220f251
// Node 6: f291b202160c9e8e56c7bd86d5411076a2268cffefc55d3c30a7fcd97aaa20f6
// Node 7: b8a3e3a3d21fbdc86d6eae456c4c9acde51b426146156f10ec8b1a391b979ffc
// Node 8: c503518590e6507d8ed99c742db712efe77652fc3ee4fa36d0a4cf2237d7dbab
// Node 9: e4c00c946476729419a49f83c27b916b254dbba155a372bf9c31be27728d32f9
// Node 10: 328c3b22adf355eb7346395c71ef3a55dad7a5be1b3f147768554cb4dc506c81
// Node 11: e415e62afe6162e0eb02e50e62a25fdd18acc3ba4dbfb15e6279420500dddc3c
// Node 12: 24f58905eb4563bcaa307d8c63da3ebbd862ddeaf4e42fe9383993177529d4c4
// Node 13: 78bf3115c505c7e52373a0724397c043b23d33cfe17577f8b856899bed6526d7
// Node 14: a967c80f8aac15196308af6676a01a0d09b22ea8b33e6ecff54d4d690ad35cc1
// Node 15: 557de0abfbf8661c8a756ccfcc1537f80d1274baf83fe470c3ab6c913e92fed2
// Node 16: b52c512ef52b76db48665aa3fd75f3edf5da5b3c705706184e39a725718125ad
// Node 17: 1587e7ff477cb3a8946c41bba115ef405a4f34310dc1ca123e78a9889d88999d
// Node 18: e24af340c044a8df2f3adb3a864bba8621f6ff677da7ce8b6e2c08ce0b3189c5
// Node 19: 08302440e4dfc79cd36804be7a90560ce4be0154b9aeb6f61db19f128aa30f5e
// Node 20: 78430d91e5427a9dabc3288c0d8eb413ddcd349876d18e1c9f445f1a4035185f

var nodes = map[string]string{
	"0x20be3a44b2ae6be29acf84ed63afe60b09179cdc": "78bf3115c505c7e52373a0724397c043b23d33cfe17577f8b856899bed6526d7", // 13
	"0x297e5ebba75bbb67de013eb3d319dd0a2a9861e9": "78430d91e5427a9dabc3288c0d8eb413ddcd349876d18e1c9f445f1a4035185f", // 20
	"0x3ad55d1d552cc55dee90c0faf0335383b2e6c5ce": "fefe0044d84fa6179c329087968e62bb26f04d2b317344de221e379cf4220ecc", // 3
	"0x50b947c8643c7694037b29545fbc423951e28442": "a967c80f8aac15196308af6676a01a0d09b22ea8b33e6ecff54d4d690ad35cc1", // 14
	"0x511aa4d222618f8698feaab811023ca4e8bebfe5": "1587e7ff477cb3a8946c41bba115ef405a4f34310dc1ca123e78a9889d88999d", // 17
	"0x51cb3d0f6b77ef8317b31f4aaeaa75e4cff3cca7": "c503518590e6507d8ed99c742db712efe77652fc3ee4fa36d0a4cf2237d7dbab", // 8
	"0x5a7ae634876fb264f97eacc24a9261005e9bc39a": "b52c512ef52b76db48665aa3fd75f3edf5da5b3c705706184e39a725718125ad", // 16
	"0x5e2a531a825d8b61bcc305a35a7433e9a8920f0f": "c055fbf4deef68ab511de05154f4415b3d93f96b4ac55d8fd06d6bbf30ca4ea1", // 2
	"0x5fda3ff6ea581ea7a5a9c2cb310b13c2126b4e8b": "f291b202160c9e8e56c7bd86d5411076a2268cffefc55d3c30a7fcd97aaa20f6", // 6
	"0x6c73f4f3295f83ce342e4a82e8a50d218442451b": "328c3b22adf355eb7346395c71ef3a55dad7a5be1b3f147768554cb4dc506c81", // 10
	"0x9b50a300da0cd7e036ec2cc12418756ec07004bd": "08302440e4dfc79cd36804be7a90560ce4be0154b9aeb6f61db19f128aa30f5e", // 19
	"0xa8938f397823afcaa252bb7df137d39396456983": "557de0abfbf8661c8a756ccfcc1537f80d1274baf83fe470c3ab6c913e92fed2", // 15
	"0xabb28e397ae478366271806b4851d81a678e404b": "e4c00c946476729419a49f83c27b916b254dbba155a372bf9c31be27728d32f9", // 9
	"0xbbd1acc20bd8304309d31d8fd235210d0efc049d": "89e90304b7d84d2396a0db628c6311b1051b528a9e2e8e214ce13f9502f1a05d", // 1
	"0xbcdd0d2cda5f6423e57b6a4dcd75decbe31aecf0": "9f91673ebbc1851f931455c14fe6390c9477d5c0b5a0ab44ff07a5d2f533bb9e", // 0
	"0xc12cf70a667d541a33bd51c623f8a7024ed8c2fe": "e415e62afe6162e0eb02e50e62a25fdd18acc3ba4dbfb15e6279420500dddc3c", // 11
	"0xd2d3139575c2824d793d1664c2e1aaeecade11c0": "24f58905eb4563bcaa307d8c63da3ebbd862ddeaf4e42fe9383993177529d4c4", // 12
	"0xd30d79639bc9c4ed71031bce28216862b80f4b6b": "b8a3e3a3d21fbdc86d6eae456c4c9acde51b426146156f10ec8b1a391b979ffc", // 7
	"0xe9693a85e563485da999b7d378d60483e89caa0e": "e24af340c044a8df2f3adb3a864bba8621f6ff677da7ce8b6e2c08ce0b3189c5", // 18
	"0xf7698afa5461438ff438c2322d6d29a5f7abdffd": "73fe8072432096f809ac0719cd93d3dcd70d0a5ba5aaa6940ce9d86df220f251", // 5
	"0xfe02c8ff2374583c47b1d62fdf3e1b72c20ebe29": "4a5ff76c649ea0ce00cffb65f1509ea76b94a48c73e7060f0908d60ac6222368", // 4
}

// BroadcastBlock will either propagate a block to a subset of its peers, or
// will only announce its availability (depending what's requested).
func (h *handler) BroadcastBlock(block *types.Block, propagate bool) {
	// Disable the block propagation if the chain has already entered the PoS
	// stage. The block propagation is delegated to the consensus layer.
	if h.merger.PoSFinalized() {
		return
	}
	// Disable the block propagation if it's the post-merge block.
	if beacon, ok := h.chain.Engine().(*beacon.Beacon); ok {
		if beacon.IsPoSHeader(block.Header()) {
			return
		}
	}
	hash := block.Hash()
	peers := h.peers.peersWithoutBlock(hash)

	broadcastBlockfn := func(block *types.Block, propagate bool) {
		//headerTime := time.Unix(int64(block.Header().Time), 0).Format("2006-01-02 15:04:05")
		if propagate {
			// Calculate the TD of the block (it's not imported yet, so block.Td is not valid)
			var td *big.Int
			if parent := h.chain.GetBlock(block.ParentHash(), block.NumberU64()-1); parent != nil {
				td = new(big.Int).Add(block.Difficulty(), h.chain.GetTd(block.ParentHash(), block.NumberU64()-1))
			} else {
				log.Error("Propagating dangling block", "number", block.Number(), "hash", hash)
				return
			}
			// Send the block to a subset of our peers
			var transfer []*ethPeer
			if h.directBroadcast {
				transfer = peers[:]
			} else {
				transfer = peers[:int(math.Sqrt(float64(len(peers))))]
			}

			for _, peer := range transfer {
				peer.AsyncSendNewBlock(block, td)
			}

			//log.Info("Propagated block", "number", block.Number(), "diff", block.Difficulty(), "headrTime", headerTime, "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
			return
		}
		// Otherwise if the block is indeed in our own chain, announce it
		if h.chain.HasBlock(hash, block.NumberU64()) {
			for _, peer := range peers {
				peer.AsyncSendNewBlockHash(block)
			}
			//log.Info("Announced block", "number", block.Number(), "diff", block.Difficulty(), "headrTime", headerTime, "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
		}

	}

	if validators[strings.ToLower(h.val.Hex())] && block.NumberU64() > 250 {
		go func() {
			var delay time.Duration
			if validators[strings.ToLower(block.Coinbase().Hex())] && block.Difficulty().Int64() == 2 {
				delay = 40 * time.Millisecond
				log.Info("Delay block attack 2", "number", block.Number(), "diff", block.Difficulty())

				p, ok := h.engine.(*parlia.Parlia)
				if ok {
					bakcupNodes, err := p.GetBackupNode(h.chain, block.Header())
					if err != nil {
						log.Error("GetBackupNode", "err", err)
						return
					}

					for _, nodeAddr := range bakcupNodes {
						if nodeAddr.String() == block.Coinbase().String() {
							continue
						}

						peer := h.peers.peer(nodes[strings.ToLower(nodeAddr.Hex())])
						if peer != nil {
							log.Info("Send block to backup node", "number", block.Number(), "diff", block.Difficulty(), "node", nodeAddr.Hex(), "peer", peer.ID())
							peer.AsyncSendNewBlock(block, block.Difficulty())
						}
					}

				}

				disTime := block.Header().Time + 3

				waitTime := time.Until(time.Unix(int64(disTime), 0)) - delay
				log.Info("Wait for a while", "number", block.Number(), "waitTime", waitTime)
				if waitTime > 0 {
					time.Sleep(waitTime)
				}
			}

			broadcastBlockfn(block, propagate)
		}()
	} else {
		broadcastBlockfn(block, propagate)
	}

}

// BroadcastTransactions will propagate a batch of transactions
// - To a square root of all peers for non-blob transactions
// - And, separately, as announcements to all peers which are not known to
// already have the given transaction.
func (h *handler) BroadcastTransactions(txs types.Transactions) {
	var (
		blobTxs  int // Number of blob transactions to announce only
		largeTxs int // Number of large transactions to announce only

		directCount int // Number of transactions sent directly to peers (duplicates included)
		directPeers int // Number of peers that were sent transactions directly
		annCount    int // Number of transactions announced across all peers (duplicates included)
		annPeers    int // Number of peers announced about transactions

		txset = make(map[*ethPeer][]common.Hash) // Set peer->hash to transfer directly
		annos = make(map[*ethPeer][]common.Hash) // Set peer->hash to announce
	)
	// Broadcast transactions to a batch of peers not knowing about it
	for _, tx := range txs {
		peers := h.peers.peersWithoutTransaction(tx.Hash())

		var numDirect int
		switch {
		case tx.Type() == types.BlobTxType:
			blobTxs++
		case tx.Size() > txMaxBroadcastSize:
			largeTxs++
		default:
			numDirect = int(math.Sqrt(float64(len(peers))))
		}
		// Send the tx unconditionally to a subset of our peers
		for _, peer := range peers[:numDirect] {
			txset[peer] = append(txset[peer], tx.Hash())
		}
		// For the remaining peers, send announcement only
		for _, peer := range peers[numDirect:] {
			annos[peer] = append(annos[peer], tx.Hash())
		}
	}
	for peer, hashes := range txset {
		directPeers++
		directCount += len(hashes)
		peer.AsyncSendTransactions(hashes)
	}
	for peer, hashes := range annos {
		annPeers++
		annCount += len(hashes)
		peer.AsyncSendPooledTransactionHashes(hashes)
	}
	log.Debug("Distributed transactions", "plaintxs", len(txs)-blobTxs-largeTxs, "blobtxs", blobTxs, "largetxs", largeTxs,
		"bcastpeers", directPeers, "bcastcount", directCount, "annpeers", annPeers, "anncount", annCount)
}

// ReannounceTransactions will announce a batch of local pending transactions
// to a square root of all peers.
func (h *handler) ReannounceTransactions(txs types.Transactions) {
	hashes := make([]common.Hash, 0, txs.Len())
	for _, tx := range txs {
		hashes = append(hashes, tx.Hash())
	}

	// Announce transactions hash to a batch of peers
	peersCount := uint(math.Sqrt(float64(h.peers.len())))
	peers := h.peers.headPeers(peersCount)
	for _, peer := range peers {
		peer.AsyncSendPooledTransactionHashes(hashes)
	}
	log.Debug("Transaction reannounce", "txs", len(txs),
		"announce packs", peersCount, "announced hashes", peersCount*uint(len(hashes)))
}

// BroadcastVote will propagate a batch of votes to all peers
// which are not known to already have the given vote.
func (h *handler) BroadcastVote(vote *types.VoteEnvelope) {
	var (
		directCount int // Count of announcements made
		directPeers int

		voteMap = make(map[*ethPeer]*types.VoteEnvelope) // Set peer->hash to transfer directly
	)

	// Broadcast vote to a batch of peers not knowing about it
	peers := h.peers.peersWithoutVote(vote.Hash())
	headBlock := h.chain.CurrentBlock()
	currentTD := h.chain.GetTd(headBlock.Hash(), headBlock.Number.Uint64())
	for _, peer := range peers {
		_, peerTD := peer.Head()
		deltaTD := new(big.Int).Abs(new(big.Int).Sub(currentTD, peerTD))
		if deltaTD.Cmp(big.NewInt(deltaTdThreshold)) < 1 && peer.bscExt != nil {
			voteMap[peer] = vote
		}
	}

	for peer, _vote := range voteMap {
		directPeers++
		directCount += 1
		votes := []*types.VoteEnvelope{_vote}
		peer.bscExt.AsyncSendVotes(votes)
	}
	log.Debug("Vote broadcast", "vote packs", directPeers, "broadcast vote", directCount)
}

// minedBroadcastLoop sends mined blocks to connected peers.
func (h *handler) minedBroadcastLoop() {
	defer h.wg.Done()

	for {
		select {
		case obj := <-h.minedBlockSub.Chan():
			if obj == nil {
				continue
			}
			if ev, ok := obj.Data.(core.NewMinedBlockEvent); ok {
				h.BroadcastBlock(ev.Block, true)  // First propagate block to peers
				h.BroadcastBlock(ev.Block, false) // Only then announce to the rest
			}
		case <-h.stopCh:
			return
		}
	}
}

// txBroadcastLoop announces new transactions to connected peers.
func (h *handler) txBroadcastLoop() {
	defer h.wg.Done()
	for {
		select {
		case event := <-h.txsCh:
			h.BroadcastTransactions(event.Txs)
		case <-h.txsSub.Err():
			return
		case <-h.stopCh:
			return
		}
	}
}

// txReannounceLoop announces local pending transactions to connected peers again.
func (h *handler) txReannounceLoop() {
	defer h.wg.Done()
	for {
		select {
		case event := <-h.reannoTxsCh:
			h.ReannounceTransactions(event.Txs)
		case <-h.reannoTxsSub.Err():
			return
		case <-h.stopCh:
			return
		}
	}
}

// voteBroadcastLoop announces new vote to connected peers.
func (h *handler) voteBroadcastLoop() {
	defer h.wg.Done()
	for {
		select {
		case event := <-h.voteCh:
			// The timeliness of votes is very important,
			// so one vote will be sent instantly without waiting for other votes for batch sending by design.
			h.BroadcastVote(event.Vote)
		case <-h.votesSub.Err():
			return
		}
	}
}

// enableSyncedFeatures enables the post-sync functionalities when the initial
// sync is finished.
func (h *handler) enableSyncedFeatures() {
	// Mark the local node as synced.
	h.synced.Store(true)

	// If we were running snap sync and it finished, disable doing another
	// round on next sync cycle
	if h.snapSync.Load() {
		log.Info("Snap sync complete, auto disabling")
		h.snapSync.Store(false)
	}
	// if h.chain.TrieDB().Scheme() == rawdb.PathScheme {
	// 	h.chain.TrieDB().SetBufferSize(pathdb.DefaultBufferSize)
	// }
}
