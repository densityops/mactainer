package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"log"

	"github.com/mikesmitty/edkey"
	"golang.org/x/crypto/ssh"
)

func GenerateKeys() (string, string, error) {
	publicKey, privateKey, err := generateKeysed25519()
	if err != nil {
		return "", "", err
	}
	publicKeyBytes, err := generatePublicKey(publicKey)
	if err != nil {
		return "", "", err
	}
	return string(encodePrivateKeyToPEM(privateKey)[:]), string(publicKeyBytes[:]), nil
}

func generateKeysed25519() (*ed25519.PublicKey, *ed25519.PrivateKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("could not generate private key: %s", err)
	}
	return &publicKey, &privateKey, nil
}

// encodePrivateKeyToPEM encodes Private Key from ed25519 to PEM format
func encodePrivateKeyToPEM(privateKey *ed25519.PrivateKey) []byte {
	// Get ASN.1 DER format
	// privDER, err := x509.MarshalPKCS8PrivateKey(*privateKey)
	// if err != nil {
	// 	return nil
	// }
	keyBytes := edkey.MarshalED25519PrivateKey(*privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: keyBytes,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)
	return privatePEM
}

// generatePublicKey take a rsa.PublicKey and return bytes suitable for writing to .pub file
// returns in the format "ssh-rsa ..."
func generatePublicKey(publickey *ed25519.PublicKey) ([]byte, error) {
	publicKey, err := ssh.NewPublicKey(*publickey)
	if err != nil {
		return nil, fmt.Errorf("could not generate public key: %s", err)
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	log.Println("Public key generated")
	return pubKeyBytes, nil
}
