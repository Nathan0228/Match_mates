package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

func loadPromoCodeKey(raw string, dev bool) ([]byte, error) {
	if raw == "" && dev {
		key := sha256.Sum256([]byte("matchmates-development-promo-code-key"))
		return key[:], nil
	}
	key, err := hex.DecodeString(raw)
	if err != nil || len(key) != 32 {
		return nil, errors.New("PROMO_CODE_KEY must be a 64-character hex string")
	}
	return key, nil
}

func (a *App) promoEncryptionKey() ([]byte, error) {
	if len(a.promoCodeKey) == 32 {
		return a.promoCodeKey, nil
	}
	return loadPromoCodeKey("", a.dev)
}

func (a *App) encryptPromoCode(id, code string) (string, error) {
	key, err := a.promoEncryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(code), []byte(id))
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (a *App) decryptPromoCode(id, encoded string) (string, error) {
	key, err := a.promoEncryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid promo code ciphertext")
	}
	plain, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], []byte(id))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
