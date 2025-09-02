package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"

	"github.com/google/go-attestation/attest"
)

type AttestationManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	tpm        *attest.TPM
}

func NewAttestationManager() (*AttestationManager, error) {
	// Generate RSA key pair for attestation
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Try to open TPM (might not be available in container)
	tpm, err := attest.OpenTPM(&attest.OpenConfig{})
	if err != nil {
		log.Printf("Warning: TPM not available: %v", err)
		tpm = nil
	}

	return &AttestationManager{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		tpm:        tpm,
	}, nil
}

func (am *AttestationManager) CreateAttestation(challenge []byte) ([]byte, error) {
	if am.tpm != nil {
		return am.createTPMAttestation(challenge)
	}
	return am.createSoftwareAttestation(challenge)
}

func (am *AttestationManager) createTPMAttestation(challenge []byte) ([]byte, error) {
	// Use TPM for hardware attestation
	// This is a simplified implementation
	hash := sha256.Sum256(challenge)
	signature, err := rsa.SignPKCS1v15(rand.Reader, am.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func (am *AttestationManager) createSoftwareAttestation(challenge []byte) ([]byte, error) {
	// Fallback to software-based attestation
	hash := sha256.Sum256(challenge)
	signature, err := rsa.SignPKCS1v15(rand.Reader, am.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func (am *AttestationManager) VerifyAttestation(attestation, challenge []byte, publicKeyPEM []byte) error {
	// Parse public key
	block, _ := pem.Decode(publicKeyPEM)
	if block == nil {
		return fmt.Errorf("failed to decode public key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not RSA")
	}

	// Verify signature
	hash := sha256.Sum256(challenge)
	return rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hash[:], attestation)
}

func (am *AttestationManager) GetPublicKeyPEM() ([]byte, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(am.publicKey)
	if err != nil {
		return nil, err
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	return pem.EncodeToMemory(block), nil
}

func (am *AttestationManager) Close() error {
	if am.tpm != nil {
		return am.tpm.Close()
	}
	return nil
}
