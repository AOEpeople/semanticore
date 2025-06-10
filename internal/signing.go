package internal

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
)

var ErrNoSigningKeyFound = errors.New("no signing key found")

// TryCreateSignKey creates a new GPG key for signing commits and tags from either the passed file
// path or if it is nil from an environment variable GPG_SIGN_KEY containing the key directly.
// If both are passed an error is returned
func TryCreateSignKey(signKeyFilePath *string) (*openpgp.Entity, error) {
	isSignKeyFilePathGiven := signKeyFilePath != nil && *signKeyFilePath != ""
	gpgKeyContent := os.Getenv("SEMANTICORE_SIGN_KEY")
	isKeyEnvVarSet := gpgKeyContent != ""

	if isSignKeyFilePathGiven && isKeyEnvVarSet {
		return nil, fmt.Errorf("[semanticore] both --sign-key-file flag and SEMANTICORE_SIGN_KEY environment variable are set. Please use only one")
	}

	if isSignKeyFilePathGiven {
		entity, err := loadGPGEntityFromFile(*signKeyFilePath)

		if err != nil {
			return nil, err
		}

		log.Printf("[semanticore] using GPG key from file: %s", *signKeyFilePath)
		return entity, nil
	}

	// GPG content
	if isKeyEnvVarSet {
		entity, err := loadGPGEntityFromContent(gpgKeyContent)

		if err != nil {
			return nil, err
		}

		log.Printf("[semanticore] using GPG key from SEMANTICORE_SIGN_KEY environment variable")
		return entity, nil
	}

	// no key available
	return nil, ErrNoSigningKeyFound
}

func loadGPGEntityFromFile(keyPath string) (*openpgp.Entity, error) {
	keyFile, err := os.Open(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open GPG key file: %w", err)
	}
	defer keyFile.Close()

	keyring, err := openpgp.ReadArmoredKeyRing(keyFile)
	if err != nil {
		// Try reading as binary format if armored reading fails
		keyFile.Seek(0, 0)
		keyring, err = openpgp.ReadKeyRing(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read GPG key: %w", err)
		}
	}

	if len(keyring) == 0 {
		return nil, fmt.Errorf("no GPG entities found in key file")
	}

	return keyring[0], nil
}

func loadGPGEntityFromContent(keyContent string) (*openpgp.Entity, error) {
	keyReader := strings.NewReader(keyContent)

	keyring, err := openpgp.ReadArmoredKeyRing(keyReader)
	if err != nil {
		// Try reading as binary format if armored reading fails
		keyReader = strings.NewReader(keyContent)
		keyring, err = openpgp.ReadKeyRing(keyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read GPG key from content: %w", err)
		}
	}

	if len(keyring) == 0 {
		return nil, fmt.Errorf("no GPG entities found in key content")
	}

	return keyring[0], nil
}
