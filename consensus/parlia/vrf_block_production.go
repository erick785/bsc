package parlia

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/vechain/go-ecvrf"
)

var (
	// errVRFNotEligible is returned when validator's VRF proof doesn't meet the threshold
	errVRFNotEligible = errors.New("validator not eligible by VRF threshold")

	// errVRFProofGeneration is returned when VRF proof generation fails
	errVRFProofGeneration = errors.New("failed to generate VRF proof")

	// errVRFProofVerification is returned when VRF proof verification fails
	errVRFProofVerification = errors.New("failed to verify VRF proof")
)

// VRFProofData contains the VRF proof components
type VRFProofData struct {
	Beta  []byte // VRF output
	Pi    []byte // VRF proof
	Alpha []byte // VRF input (block number)
}

// generateVRFProof generates a VRF proof for the given block number using validator's private key
func generateVRFProof(privateKey *ecdsa.PrivateKey, blockNumber *big.Int) (*VRFProofData, error) {
	if privateKey == nil {
		return nil, errors.New("private key is nil")
	}

	// Alpha is the block number
	alpha := blockNumber.Bytes()

	// Generate VRF proof using secp256k1 curve
	beta, pi, err := ecvrf.Secp256k1Sha256Tai.Prove(privateKey, alpha)
	if err != nil {
		return nil, err
	}

	return &VRFProofData{
		Beta:  beta,
		Pi:    pi,
		Alpha: alpha,
	}, nil
}

// verifyVRFProof verifies a VRF proof
func verifyVRFProof(publicKey *ecdsa.PublicKey, blockNumber *big.Int, pi []byte) ([]byte, error) {
	if publicKey == nil {
		return nil, errors.New("public key is nil")
	}

	alpha := blockNumber.Bytes()

	// Verify VRF proof
	beta, err := ecvrf.Secp256k1Sha256Tai.Verify(publicKey, alpha, pi)
	if err != nil {
		return nil, err
	}

	return beta, nil
}

// hexToDigit converts the first hex character to its numeric value (0-15)
func hexToDigit(hexByte byte) uint8 {
	if hexByte >= '0' && hexByte <= '9' {
		return hexByte - '0'
	}
	if hexByte >= 'a' && hexByte <= 'f' {
		return hexByte - 'a' + 10
	}
	if hexByte >= 'A' && hexByte <= 'F' {
		return hexByte - 'A' + 10
	}
	return 16 // Invalid, won't pass threshold
}

// getFirstHexDigit extracts the first hex digit from beta value
func getFirstHexDigit(beta []byte) uint8 {
	if len(beta) == 0 {
		return 16 // Invalid
	}

	// Convert beta to hex string and get first character
	hexStr := common.Bytes2Hex(beta)
	if len(hexStr) == 0 {
		return 16
	}

	return hexToDigit(hexStr[0])
}

func getDifficultyHexDigit(beta []byte) *big.Int {
	if len(beta) == 0 {
		return big.NewInt(16 * 4)
	}

	// Convert beta to hex string and get first character
	hexStr := common.Bytes2Hex(beta)
	if len(hexStr) == 0 {
		return big.NewInt(16 * 4)
	}

	d := hexToDigit(hexStr[0]) + hexToDigit(hexStr[1]) + hexToDigit(hexStr[2]) + hexToDigit(hexStr[3])
	return big.NewInt(int64(d))
}

// getVRFThreshold calculates the dynamic VRF threshold based on time since parent block
// The threshold decreases over time to ensure liveness (logic: firstDigit > threshold to be eligible)
func (p *Parlia) getVRFThreshold(blockNumber uint64, timeSinceParent int64) uint8 {
	// If VRF is not enabled, return 0 threshold (all validators eligible since any digit > 0)
	if !p.config.EnableVRF {
		return 0
	}

	// Before VRF activation, return 0 threshold (all validators eligible)
	if p.config.VRFActivationBlock == nil || blockNumber < p.config.VRFActivationBlock.Uint64() {
		return 0
	}

	// Get base threshold from config (default 8)
	baseThreshold := p.config.VRFBaseThreshold
	if baseThreshold == 0 {
		baseThreshold = 8
	}

	// Get degrade interval from config (default period)
	degradeInterval := p.config.VRFDegradeInterval
	if degradeInterval == 0 {
		degradeInterval = p.config.Period
	}

	return baseThreshold

	// todo
	// Calculate degraded threshold based on time
	// Each degradeInterval seconds, decrease threshold to allow more validators
	// Logic: firstDigit > threshold, so lower threshold = more validators eligible
	if timeSinceParent <= int64(degradeInterval) {
		return baseThreshold // Normal case: first digit > 8
	} else if timeSinceParent <= int64(degradeInterval*2) {
		return max(baseThreshold-2, 0) // first digit > 6
	} else if timeSinceParent <= int64(degradeInterval*3) {
		return max(baseThreshold-4, 0) // first digit > 4
	} else {
		return 0 // All validators eligible (any first digit > 0 works, so 1-f all eligible)
	}
}

// max returns the maximum of two uint8 values
func max(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of two uint8 values
func min(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}

// checkVRFEligibility checks if a validator is eligible to produce a block based on VRF
func (p *Parlia) checkVRFEligibility(privateKey *ecdsa.PrivateKey, header *types.Header, parentTime uint64) (bool, *VRFProofData, error) {
	blockNumber := header.Number.Uint64()

	// If VRF is not enabled, all validators are eligible
	if !p.config.EnableVRF {
		log.Debug("VRF not enabled, validator eligible by default",
			"blockNumber", blockNumber)
		return true, nil, nil
	}

	// Before VRF activation block, all validators are eligible
	if p.config.VRFActivationBlock == nil || blockNumber < p.config.VRFActivationBlock.Uint64() {
		log.Info("VRF not yet activated, validator eligible by default",
			"blockNumber", blockNumber,
			"vrfActivationBlock", p.config.VRFActivationBlock)
		return true, nil, nil
	}

	log.Info("VRF IS ACTIVE - checking eligibility",
		"blockNumber", blockNumber,
		"vrfActivationBlock", p.config.VRFActivationBlock,
		"hasPrivateKey", privateKey != nil)

	// Generate VRF proof
	vrfProof, err := generateVRFProof(privateKey, header.Number)
	if err != nil {
		log.Error("Failed to generate VRF proof", "blockNumber", blockNumber, "err", err)
		return false, nil, errVRFProofGeneration
	}

	// Get first hex digit from beta
	firstDigit := getFirstHexDigit(vrfProof.Beta)

	// Calculate current threshold
	timeSinceParent := int64(header.Time - parentTime)
	threshold := p.getVRFThreshold(blockNumber, timeSinceParent)

	// Check eligibility (changed: now firstDigit > threshold to be eligible)
	eligible := firstDigit > threshold

	if eligible {
		log.Warn("✅ VRF ELIGIBLE - Validator CAN produce block",
			"blockNumber", blockNumber,
			"beta", common.Bytes2Hex(vrfProof.Beta),
			"firstDigit", firstDigit,
			"threshold", threshold,
			"timeSinceParent", timeSinceParent,
			"condition", fmt.Sprintf("%d > %d", firstDigit, threshold))
	} else {
		log.Warn("❌ VRF NOT ELIGIBLE - Validator CANNOT produce block",
			"blockNumber", blockNumber,
			"beta", common.Bytes2Hex(vrfProof.Beta),
			"firstDigit", firstDigit,
			"threshold", threshold,
			"timeSinceParent", timeSinceParent,
			"condition", fmt.Sprintf("%d > %d is FALSE", firstDigit, threshold))
	}

	return eligible, vrfProof, nil
}

// encodeVRFProof encodes VRF proof data to bytes for storage in header
func encodeVRFProof(proof *VRFProofData) []byte {
	if proof == nil {
		return nil
	}

	// Format: [beta_length(2 bytes)][beta][pi_length(2 bytes)][pi]
	betaLen := uint16(len(proof.Beta))
	piLen := uint16(len(proof.Pi))

	result := make([]byte, 0, 4+len(proof.Beta)+len(proof.Pi))

	// Encode beta length and data
	result = append(result, byte(betaLen>>8), byte(betaLen))
	result = append(result, proof.Beta...)

	// Encode pi length and data
	result = append(result, byte(piLen>>8), byte(piLen))
	result = append(result, proof.Pi...)

	return result
}

// decodeVRFProof decodes VRF proof data from bytes
func decodeVRFProof(data []byte) (*VRFProofData, error) {
	if len(data) < 4 {
		return nil, errors.New("VRF proof data too short")
	}

	offset := 0

	// Decode beta
	betaLen := uint16(data[offset])<<8 | uint16(data[offset+1])
	offset += 2
	if offset+int(betaLen) > len(data) {
		return nil, errors.New("invalid VRF proof: beta length exceeds data")
	}
	beta := data[offset : offset+int(betaLen)]
	offset += int(betaLen)

	// Decode pi
	if offset+2 > len(data) {
		return nil, errors.New("invalid VRF proof: missing pi length")
	}
	piLen := uint16(data[offset])<<8 | uint16(data[offset+1])
	offset += 2
	if offset+int(piLen) > len(data) {
		return nil, errors.New("invalid VRF proof: pi length exceeds data")
	}
	pi := data[offset : offset+int(piLen)]

	return &VRFProofData{
		Beta: beta,
		Pi:   pi,
	}, nil
}

// calculateSealDelay calculates the delay before sealing a block
// If VRF is enabled and active, all eligible validators produce blocks at the same time (no backoff)
// Otherwise, use the traditional in-turn/out-of-turn delay mechanism
func (p *Parlia) calculateSealDelay(snap *Snapshot, header *types.Header, parent *types.Header) time.Duration {
	// VRF logic is already handled in delayForRamanujanFork
	return p.delayForRamanujanFork(snap, header)
}

// verifyVRFEligibility verifies if the block producer was eligible based on VRF
func (p *Parlia) verifyVRFEligibility(header *types.Header, parentTime uint64) error {
	// If VRF is not enabled, skip verification
	if !p.config.EnableVRF {
		return nil
	}

	// Before VRF activation block, skip verification
	blockNumber := header.Number.Uint64()
	if p.config.VRFActivationBlock == nil || blockNumber < p.config.VRFActivationBlock.Uint64() {
		return nil
	}

	// Extract VRF proof from header.VRFProof field
	if len(header.VRFProof) == 0 {
		return errors.New("missing VRF proof in header")
	}

	vrfProof, err := decodeVRFProof(header.VRFProof)
	if err != nil {
		return err
	}

	// Get validator's public key from header signature
	if len(header.Extra) < extraSeal {
		return errors.New("missing signature in header")
	}
	signature := header.Extra[len(header.Extra)-extraSeal:]

	// Recover the public key from signature
	pubkey, err := crypto.SigToPub(types.SealHash(header, p.chainConfig.ChainID).Bytes(), signature)
	if err != nil {
		return err
	}

	// Verify VRF proof
	beta, err := verifyVRFProof(pubkey, header.Number, vrfProof.Pi)
	if err != nil {
		return errVRFProofVerification
	}

	// Verify beta matches
	if !bytes.Equal(beta, vrfProof.Beta) {
		return errors.New("VRF beta mismatch")
	}

	// Check eligibility based on first hex digit (changed: now firstDigit > threshold to be eligible)
	firstDigit := getFirstHexDigit(beta)
	timeSinceParent := int64(header.Time - parentTime)
	threshold := p.getVRFThreshold(blockNumber, timeSinceParent)

	if firstDigit <= threshold {
		log.Warn("VRF eligibility verification failed",
			"blockNumber", blockNumber,
			"coinbase", header.Coinbase.Hex(),
			"beta", common.Bytes2Hex(beta),
			"firstDigit", firstDigit,
			"threshold", threshold)
		return errVRFNotEligible
	}

	log.Debug("VRF eligibility verified",
		"blockNumber", blockNumber,
		"coinbase", header.Coinbase.Hex(),
		"firstDigit", firstDigit,
		"threshold", threshold)

	return nil
}
