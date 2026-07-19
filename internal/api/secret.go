package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

type secretCodec struct{ key []byte }

func newSecretCodec(secret string) secretCodec {
	if secret == "" {
		return secretCodec{}
	}
	sum := sha256.Sum256([]byte(secret))
	return secretCodec{key: sum[:]}
}

func (c secretCodec) Encrypt(value string) (string, error) {
	if len(c.key) == 0 {
		return "", errors.New("APP_SECRET_KEY 未配置，不能保存渠道密钥")
	}
	block, err := aes.NewCipher(c.key)
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
	sealed := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (c secretCodec) Decrypt(encoded string) (string, error) {
	if len(c.key) == 0 {
		return "", errors.New("APP_SECRET_KEY 未配置")
	}
	sealed, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.New("密钥数据无效")
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(sealed) < gcm.NonceSize() {
		return "", errors.New("密钥数据无效")
	}
	plaintext, err := gcm.Open(nil, sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():], nil)
	if err != nil {
		return "", errors.New("无法解密密钥数据")
	}
	return string(plaintext), nil
}

func secretHint(value string) string {
	if len(value) <= 4 {
		return "••••"
	}
	return "••••" + value[len(value)-4:]
}
