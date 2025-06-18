// Package wireguard provides WireGuard VPN functionality including cryptographic key management,
// configuration file generation, and server control operations. This package handles the
// low-level WireGuard protocol operations and integrates with the operating system's
// WireGuard implementation.
package wireguard

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	
	"golang.org/x/crypto/curve25519"
)

// KeyPair represents a WireGuard cryptographic key pair.
// It contains both the private and public keys in base64-encoded format,
// as required by the WireGuard configuration format. The keys are generated
// using Curve25519 elliptic curve cryptography for optimal security and performance.
type KeyPair struct {
	PrivateKey string // Base64-encoded private key (32 bytes)
	PublicKey  string // Base64-encoded public key (32 bytes)
}

// GenerateKeyPair creates a new cryptographically secure WireGuard key pair.
// It uses the system's cryptographically secure random number generator
// to create a private key, then derives the corresponding public key using
// Curve25519 elliptic curve operations. Both keys are encoded in base64 format
// for compatibility with WireGuard configuration files.
// Returns a KeyPair pointer or an error if key generation fails.
func GenerateKeyPair() (*KeyPair, error) {
	var private [32]byte
	_, err := rand.Read(private[:])
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	public, err := curve25519.X25519(private[:], curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	return &KeyPair{
		PrivateKey: base64.StdEncoding.EncodeToString(private[:]),
		PublicKey:  base64.StdEncoding.EncodeToString(public),
	}, nil
}

// PrivateKeyBytes decodes the base64-encoded private key and returns it as a byte array.
// This method is useful when the raw key bytes are needed for cryptographic operations
// or when interfacing with lower-level WireGuard APIs that expect binary key data.
// Returns a 32-byte array containing the private key or an error if decoding fails.
func (kp *KeyPair) PrivateKeyBytes() ([32]byte, error) {
	var key [32]byte
	decoded, err := base64.StdEncoding.DecodeString(kp.PrivateKey)
	if err != nil {
		return key, err
	}
	copy(key[:], decoded)
	return key, nil
}

// PublicKeyBytes decodes the base64-encoded public key and returns it as a byte array.
// This method is useful when the raw key bytes are needed for cryptographic operations
// or when interfacing with lower-level WireGuard APIs that expect binary key data.
// Returns a 32-byte array containing the public key or an error if decoding fails.
func (kp *KeyPair) PublicKeyBytes() ([32]byte, error) {
	var key [32]byte
	decoded, err := base64.StdEncoding.DecodeString(kp.PublicKey)
	if err != nil {
		return key, err
	}
	copy(key[:], decoded)
	return key, nil
}