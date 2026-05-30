package downloader

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	downloaderSplitStartHeight uint64 = params.NetworkSplitStartHeight
	downloaderSplitEndHeight   uint64 = params.NetworkSplitEndHeight
)

func downloaderIsNetworkSplit(blockNumber uint64) bool {
	return blockNumber >= downloaderSplitStartHeight && blockNumber < downloaderSplitEndHeight
}

func downloaderGetValidatorGroup(addr common.Address) string {
	return params.NetworkSplitValidatorGroup(addr)
}

func downloaderIsCompatibleGroup(a, b string) bool {
	return params.NetworkSplitGroupsCompatible(a, b)
}

// downloaderFilterPartitionBlocks removes blocks mined by cross-group validators
// during the network split window. localValidator is this node's validator address.
func downloaderFilterPartitionBlocks(localValidator common.Address, blocks []*types.Block, hasBlock func(common.Hash, uint64) bool) []*types.Block {
	myGroup := downloaderGetValidatorGroup(localValidator)
	if myGroup == "" {
		// Not a known validator, no filtering.
		return blocks
	}

	filtered := blocks[:0]
	for _, b := range blocks {
		if !downloaderIsNetworkSplit(b.NumberU64()) {
			filtered = append(filtered, b)
			continue
		}
		parentKnown := b.NumberU64() == 0 || hasBlock(b.ParentHash(), b.NumberU64()-1)
		if !parentKnown && len(filtered) > 0 {
			parent := filtered[len(filtered)-1]
			parentKnown = parent.Hash() == b.ParentHash() && parent.NumberU64()+1 == b.NumberU64()
		}
		if !parentKnown {
			log.Debug("[Partition] downloader: dropping block with unknown parent",
				"number", b.NumberU64(),
				"hash", b.Hash(),
				"miner", b.Coinbase(),
				"parent", b.ParentHash(),
				"myGroup", myGroup,
			)
			continue
		}
		minerGroup := downloaderGetValidatorGroup(b.Coinbase())
		if downloaderIsCompatibleGroup(myGroup, minerGroup) {
			filtered = append(filtered, b)
		} else {
			log.Debug("[Partition] downloader: dropping cross-group block",
				"number", b.NumberU64(),
				"hash", b.Hash(),
				"miner", b.Coinbase(),
				"minerGroup", minerGroup,
				"myGroup", myGroup,
			)

		}
	}
	return filtered
}
