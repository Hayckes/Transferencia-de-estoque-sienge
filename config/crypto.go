package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	SecretKeyFileName = "secret.key"
	keySize           = 32
	nonceSize         = 12
	cipherPrefix      = "v1:"
)

var ErrInvalidSecretKey = errors.New("secret.key invalido")

func (s Store) LoadOrCreateKey() ([]byte, error) {
	if err := s.EnsureDir(); err != nil {
		return nil, err
	}

	key, err := os.ReadFile(s.SecretKeyPath())
	if errors.Is(err, os.ErrNotExist) {
		return s.createKey()
	}
	if err != nil {
		return nil, err
	}
	if len(key) != keySize {
		return nil, fmt.Errorf("%w: tamanho esperado de 32 bytes", ErrInvalidSecretKey)
	}

	return key, nil
}

func (s Store) createKey() ([]byte, error) {
	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	if err := os.WriteFile(s.SecretKeyPath(), key, 0o600); err != nil {
		return nil, err
	}

	return key, nil
}

func EncryptString(plaintext string, key []byte) (string, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, ciphertext...)

	return cipherPrefix + base64.StdEncoding.EncodeToString(payload), nil
}

func DecryptString(encrypted string, key []byte) (string, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(encrypted, cipherPrefix) {
		return "", errors.New("texto criptografado sem versao reconhecida")
	}

	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encrypted, cipherPrefix))
	if err != nil {
		return "", err
	}
	if len(payload) <= nonceSize {
		return "", errors.New("texto criptografado invalido")
	}

	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != keySize {
		return nil, fmt.Errorf("%w: tamanho esperado de 32 bytes", ErrInvalidSecretKey)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}
