package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/vechain/go-ecvrf"
)

func main() {
	fmt.Println("Hello, World!")

	sk, err := crypto.GenerateKey()
	if err != nil {
		fmt.Printf("Failed to generate private key: %v", err)
		return
	}
	//fmt.Printf("Private key: %v", pk)

	alpha := "Hello VeChain"
	beta, pi, err := ecvrf.Secp256k1Sha256Tai.Prove(sk, []byte(alpha))
	if err != nil {
		// something wrong.
		// most likely sk is not properly loaded.
		return
	}

	fmt.Println("Beta:", common.Bytes2Hex(beta))
	fmt.Println("Pi:", common.Bytes2Hex(pi))

	pk := sk.PublicKey

	// `pi` is the VRF proof
	beta, err = ecvrf.Secp256k1Sha256Tai.Verify(&pk, []byte(alpha), pi)
	if err != nil {
		// invalid proof
		return
	}

	fmt.Println("Beta:", common.Bytes2Hex(beta))
	fmt.Println("Pi:", common.Bytes2Hex(pi))

}
