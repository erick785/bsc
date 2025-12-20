// Copyright 2021 The go-ethereum Authors
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

package core

import (
	"bytes"
	crand "crypto/rand"
	"errors"
	"math/big"
	mrand "math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// ChainReader defines a small collection of methods needed to access the local
// blockchain during header verification. It's implemented by both blockchain
// and lightchain.
type ChainReader interface {
	// Config retrieves the header chain's chain configuration.
	Config() *params.ChainConfig

	// Engine retrieves the blockchain's consensus engine.
	Engine() consensus.Engine

	// GetJustifiedNumber returns the highest justified blockNumber on the branch including and before `header`
	GetJustifiedNumber(header *types.Header) uint64

	// GetTd returns the total difficulty of a local block.
	GetTd(common.Hash, uint64) *big.Int
}

// ForkChoice is the fork chooser based on the highest total difficulty of the
// chain(the fork choice used in the eth1) and the external fork choice (the fork
// choice used in the eth2). This main goal of this ForkChoice is not only for
// offering fork choice during the eth1/2 merge phase, but also keep the compatibility
// for all other proof-of-work networks.
type ForkChoice struct {
	chain ChainReader
	rand  *mrand.Rand

	// preserve is a helper function used in td fork choice.
	// Miners will prefer to choose the local mined block if the
	// local td is equal to the extern one. It can be nil for light
	// client
	preserve func(header *types.Header) bool
}

func NewForkChoice(chainReader ChainReader, preserve func(header *types.Header) bool) *ForkChoice {
	// Seed a fast but crypto originating random generator
	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		log.Crit("Failed to initialize random seed", "err", err)
	}
	return &ForkChoice{
		chain:    chainReader,
		rand:     mrand.New(mrand.NewSource(seed.Int64())),
		preserve: preserve,
	}
}

// getVRFBeta extracts the VRF beta value from a header's Extra field
// Returns nil if VRF proof is missing or invalid
func getVRFBeta(header *types.Header) []byte {
	// Extract VRF proof from Extra field
	// Extra format: [vanity 32][validators (variable, epoch only)][vrfLen 2][vrfProof (variable)][signature 65]
	const (
		extraVanity    = 32
		extraSeal      = 65
		vrfProofLength = 2
	)

	extra := header.Extra
	if len(extra) < extraVanity+extraSeal {
		return nil
	}

	// Find VRF proof start position (after vanity and validators if epoch)
	// For simplicity, we scan for VRF proof length marker before signature
	if len(extra) <= extraVanity+extraSeal+vrfProofLength {
		return nil // No room for VRF proof
	}

	// VRF proof is located somewhere before the signature
	// Search backwards to find it
	endSearchPos := len(extra) - extraSeal

	// Try to find VRF proof by checking backwards from signature
	var vrfProofData []byte
	for vrfLenPos := endSearchPos - vrfProofLength; vrfLenPos >= extraVanity; vrfLenPos-- {
		vrfLen := int(extra[vrfLenPos])<<8 | int(extra[vrfLenPos+1])

		// Check if this could be a valid VRF length
		if vrfLen > 0 && vrfLen <= 1024 {
			vrfProofStart := vrfLenPos + vrfProofLength
			vrfProofEnd := vrfProofStart + vrfLen

			// Check if this structure fits perfectly before the signature
			if vrfProofEnd+extraSeal == len(extra) {
				vrfProofData = extra[vrfProofStart:vrfProofEnd]
				break
			}
		}

		// Optimization: don't search too far back
		if vrfLenPos < endSearchPos-1024-vrfProofLength {
			break
		}
	}

	if vrfProofData == nil {
		return nil
	}

	// Decode VRF proof to extract beta value
	// Format: [beta_length(2 bytes)][beta][pi_length(2 bytes)][pi]
	if len(vrfProofData) < 4 {
		return nil
	}

	// Extract beta length
	betaLen := uint16(vrfProofData[0])<<8 | uint16(vrfProofData[1])
	if len(vrfProofData) < int(2+betaLen) {
		return nil
	}

	// Extract beta
	beta := vrfProofData[2 : 2+betaLen]
	return beta
}

// isVRFActive checks if VRF-based fork choice is active for the given block
func (f *ForkChoice) isVRFActive(blockNumber uint64) bool {
	config := f.chain.Config()
	if config.Parlia == nil || !config.Parlia.EnableVRF {
		return false
	}
	if config.Parlia.VRFActivationBlock == nil {
		return false
	}
	return blockNumber >= config.Parlia.VRFActivationBlock.Uint64()
}

// reorgNeeded returns whether the reorg should be applied
// based on the given external header and local canonical chain.
// In the td mode, the new head is chosen if the corresponding
// total difficulty is higher. In the extern mode, the trusted
// header is always selected as the head.
func (f *ForkChoice) ReorgNeeded(current *types.Header, extern *types.Header) (bool, error) {
	var (
		localTD  = f.chain.GetTd(current.Hash(), current.Number.Uint64())
		externTd = f.chain.GetTd(extern.Hash(), extern.Number.Uint64())
	)
	if localTD == nil {
		return false, errors.New("missing td")
	}
	if externTd == nil {
		ptd := f.chain.GetTd(extern.ParentHash, extern.Number.Uint64()-1)
		if ptd == nil {
			return false, consensus.ErrUnknownAncestor
		}
		externTd = new(big.Int).Add(ptd, extern.Difficulty)
	}
	// Accept the new header as the chain head if the transition
	// is already triggered. We assume all the headers after the
	// transition come from the trusted consensus layer.
	if ttd := f.chain.Config().TerminalTotalDifficulty; ttd != nil && ttd.Cmp(externTd) <= 0 {
		return true, nil
	}

	// If the total difficulty is higher than our known, add it to the canonical chain
	if diff := externTd.Cmp(localTD); diff > 0 {
		return true, nil
	} else if diff < 0 {
		return false, nil
	}
	// Local and external difficulty is identical.
	// Second clause in the if statement reduces the vulnerability to selfish mining.
	// Please refer to http://www.cs.cornell.edu/~ie53/publications/btcProcFC.pdf
	reorg := false
	externNum, localNum := extern.Number.Uint64(), current.Number.Uint64()
	if externNum < localNum {
		reorg = true
	} else if externNum == localNum {
		// VRF-based fork choice for blocks at the same height
		// If VRF is active, compare beta values; higher beta wins (changed from lower to higher)
		if f.isVRFActive(externNum) {
			currentBeta := getVRFBeta(current)
			externBeta := getVRFBeta(extern)

			// If both headers have valid VRF proofs, use VRF-based comparison
			if currentBeta != nil && externBeta != nil {
				cmp := bytes.Compare(externBeta, currentBeta)
				if cmp > 0 {
					// extern beta is larger, should reorg
					log.Info("VRF-based fork choice: extern has higher beta",
						"currentBeta", common.Bytes2Hex(currentBeta),
						"externBeta", common.Bytes2Hex(externBeta),
						"currentNum", current.Number,
						"currentHash", current.Hash(),
						"externNum", extern.Number,
						"externHash", extern.Hash())
					return true, nil
				} else if cmp < 0 {
					// current beta is larger, keep current
					log.Info("VRF-based fork choice: current has higher beta",
						"currentBeta", common.Bytes2Hex(currentBeta),
						"externBeta", common.Bytes2Hex(externBeta),
						"currentNum", current.Number,
						"currentHash", current.Hash(),
						"externNum", extern.Number,
						"externHash", extern.Hash())
					return false, nil
				}
				// If beta values are equal (very unlikely), fall through to other rules
				log.Warn("VRF beta values are identical, using fallback rules",
					"beta", common.Bytes2Hex(currentBeta),
					"currentNum", current.Number,
					"externNum", extern.Number)
			} else {
				// Log if VRF proofs are missing (shouldn't happen in VRF mode)
				if currentBeta == nil {
					log.Debug("Current header missing VRF beta", "number", current.Number, "hash", current.Hash())
				}
				if externBeta == nil {
					log.Debug("Extern header missing VRF beta", "number", extern.Number, "hash", extern.Hash())
				}
			}
		}

		// Fallback to traditional fork choice rules
		var currentPreserve, externPreserve bool
		if f.preserve != nil {
			currentPreserve, externPreserve = f.preserve(current), f.preserve(extern)
		}
		choiceRules := func() bool {
			if extern.Time == current.Time {
				doubleSign := (extern.Coinbase == current.Coinbase)
				if doubleSign {
					return extern.Hash().Cmp(current.Hash()) < 0
				} else {
					return f.rand.Float64() < 0.5
				}
			} else {
				return extern.Time < current.Time
			}
		}
		reorg = !currentPreserve && (externPreserve || choiceRules())
	}
	return reorg, nil
}

// ReorgNeededWithFastFinality compares justified block numbers firstly, backoff to compare tds when equal
func (f *ForkChoice) ReorgNeededWithFastFinality(current *types.Header, header *types.Header) (bool, error) {
	_, ok := f.chain.Engine().(consensus.PoSA)
	if !ok {
		return f.ReorgNeeded(current, header)
	}

	justifiedNumber, curJustifiedNumber := uint64(0), uint64(0)
	if f.chain.Config().IsPlato(header.Number) {
		justifiedNumber = f.chain.GetJustifiedNumber(header)
	}
	if f.chain.Config().IsPlato(current.Number) {
		curJustifiedNumber = f.chain.GetJustifiedNumber(current)
	}
	if justifiedNumber == curJustifiedNumber {
		return f.ReorgNeeded(current, header)
	}

	if justifiedNumber > curJustifiedNumber && header.Number.Cmp(current.Number) <= 0 {
		log.Info("Chain find higher justifiedNumber", "fromHeight", current.Number, "fromHash", current.Hash(), "fromMiner", current.Coinbase, "fromJustified", curJustifiedNumber,
			"toHeight", header.Number, "toHash", header.Hash(), "toMiner", header.Coinbase, "toJustified", justifiedNumber)
	}
	return justifiedNumber > curJustifiedNumber, nil
}
