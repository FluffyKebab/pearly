package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

type Encrypter interface {
	Encrypt([]byte) ([]byte, error)
}

type Decrypter interface {
	Decrypt([]byte) ([]byte, error)
}

type EncryptDecrypter interface {
	Encrypter
	Decrypter
}

type symetricEncryptor struct {
	aead cipher.AEAD
}

func NewSymetricEncryption(secretKey []byte) (symetricEncryptor, error) {
	aes, err := aes.NewCipher(secretKey)
	if err != nil {
		return symetricEncryptor{}, err
	}

	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		return symetricEncryptor{}, err
	}

	return symetricEncryptor{gcm}, nil
}

func (e symetricEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.aead.NonceSize())
	_, err := rand.Read(nonce)
	if err != nil {
		return nil, err
	}

	return e.aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (e symetricEncryptor) Decrypt(chiper []byte) ([]byte, error) {
	if len(chiper) < e.aead.NonceSize() {
		return nil, errors.New("chipertext missing nonce")
	}

	nonce := chiper[:e.aead.NonceSize()]
	chipertext := chiper[e.aead.NonceSize():]
	return e.aead.Open(nil, nonce, chipertext, nil)
}

func NewSymetricEncryptionSecretKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

type asymetricEncryptor struct {
}

func (e asymetricEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	return nil, nil
}
