package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"strings"
)

func getEncryptionKey() []byte {
	key := os.Getenv("APP_ENCRYPTION_KEY")
	if key == "" {
		return nil
	}
	decoded, err := hex.DecodeString(key)
	if err != nil || len(decoded) != 32 {
		return nil
	}
	return decoded
}

func EncryptPass(plaintext string) string {
	if plaintext == "" {
		return ""
	}
	key := getEncryptionKey()
	if key == nil {
		return plaintext
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return plaintext
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return plaintext
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return plaintext
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return "enc:" + base64.StdEncoding.EncodeToString(ciphertext)
}

func DecryptPass(encoded string) string {
	if encoded == "" || !strings.HasPrefix(encoded, "enc:") {
		return encoded
	}
	key := getEncryptionKey()
	if key == nil {
		return strings.TrimPrefix(encoded, "enc:")
	}
	data, err := base64.StdEncoding.DecodeString(encoded[4:])
	if err != nil {
		return encoded
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return encoded
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return encoded
	}
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return encoded
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return encoded
	}
	return string(plaintext)
}
