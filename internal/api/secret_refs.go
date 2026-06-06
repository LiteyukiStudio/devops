package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"os"
	"strings"
)

const (
	encryptedSecretRefPrefix = "secret:v1:"
	literalSecretRefPrefix   = "literal:"
)

func storedSecretRef(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	block, err := aes.NewCipher(secretRefKey())
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return ""
	}
	payload := append(nonce, gcm.Seal(nil, nonce, []byte(secret), nil)...)
	return encryptedSecretRefPrefix + base64.RawURLEncoding.EncodeToString(payload)
}

func resolveStoredSecretRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if strings.HasPrefix(ref, encryptedSecretRefPrefix) {
		payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(ref, encryptedSecretRefPrefix))
		if err != nil {
			return ""
		}
		block, err := aes.NewCipher(secretRefKey())
		if err != nil {
			return ""
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return ""
		}
		if len(payload) < gcm.NonceSize() {
			return ""
		}
		nonce := payload[:gcm.NonceSize()]
		ciphertext := payload[gcm.NonceSize():]
		secret, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return ""
		}
		return string(secret)
	}
	if strings.HasPrefix(ref, literalSecretRefPrefix) {
		return strings.TrimPrefix(ref, literalSecretRefPrefix)
	}
	return ref
}

func secretRefHasValue(ref string) bool {
	return strings.TrimSpace(ref) != ""
}

func safeClientSecretRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if strings.HasPrefix(ref, encryptedSecretRefPrefix) || strings.HasPrefix(ref, literalSecretRefPrefix) {
		return ""
	}
	return ref
}

func secretRefKey() []byte {
	keyMaterial := strings.TrimSpace(os.Getenv("SECRET_ENCRYPTION_KEY"))
	if keyMaterial == "" {
		keyMaterial = "liteyuki-devops-local-secret"
	}
	sum := sha256.Sum256([]byte(keyMaterial))
	return sum[:]
}
