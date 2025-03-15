package crypto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSymetricalEncryption(t *testing.T) {
	encryptor, err := NewSymetricEncryption(NewSymetricEncryptionSecretKey())
	require.NoError(t, err)

	msg := "halo this is the mesage that should be the same anyways"
	chiper, err := encryptor.Encrypt([]byte(msg))
	require.NoError(t, err)

	plaintext, err := encryptor.Decrypt(chiper)
	require.NoError(t, err)
	require.Equal(t, msg, string(plaintext))
}
