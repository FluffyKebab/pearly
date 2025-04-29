package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
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

type symmetricEncrypter struct {
	aead cipher.AEAD
}

func NewSymmetricEncryption(secretKey []byte) (symmetricEncrypter, error) {
	aes, err := aes.NewCipher(secretKey)
	if err != nil {
		return symmetricEncrypter{}, err
	}

	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		return symmetricEncrypter{}, err
	}

	return symmetricEncrypter{gcm}, nil
}

func (e symmetricEncrypter) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.aead.NonceSize())
	_, err := rand.Read(nonce)
	if err != nil {
		return nil, err
	}

	return e.aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (e symmetricEncrypter) Decrypt(cipher []byte) ([]byte, error) {
	if len(cipher) < e.aead.NonceSize() {
		return nil, errors.New("cipher missing nonce")
	}

	nonce := cipher[:e.aead.NonceSize()]
	ciphertext := cipher[e.aead.NonceSize():]
	return e.aead.Open(nil, nonce, ciphertext, nil)
}

func NewSymmetricEncryptionSecretKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

type stream struct {
	io.Reader
	io.Writer
}

func NewEncryptionStream(secretKey []byte, underlying io.ReadWriter) (*stream, error) {
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return nil, err
	}

	var iv [aes.BlockSize]byte
	rs := cipher.NewCTR(block, iv[:])
	ws := cipher.NewCTR(block, iv[:])
	return &stream{
		Reader: cipher.StreamReader{S: rs, R: underlying},
		Writer: cipher.StreamWriter{S: ws, W: underlying},
	}, nil

}
