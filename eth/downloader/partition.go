package downloader

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// downloaderIsNetworkSplit reports whether blockNumber lies in the experiment window.
func downloaderIsNetworkSplit(blockNumber uint64) bool {
	return blockNumber >= params.NetworkSplitStartHeight && blockNumber < params.NetworkSplitEndHeight
}

// downloaderFilterPartitionBlocks is intentionally a no-op for this experiment.
//
// The experiment needs parent blocks to be retrievable on demand so that nodes can reorg
// onto the opposing chain (e.g. 0x5f fetching 400, 401, 402 from 0xbb once it receives
// block 402). Filtering cross-chain blocks here would prevent those reorgs from succeeding.
//
// Block-level routing restrictions are applied only at broadcast time by
// (*handler).filterBroadcastPeersByNetworkSplit in eth/handler.go.
func downloaderFilterPartitionBlocks(localValidator common.Address, blocks []*types.Block) []*types.Block {
	return blocks
}
