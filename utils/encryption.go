package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"theransticslabs/m/config"
)

// getCipher creates a CBC cipher using the provided base64-encoded key.
func getCipher(key string) (cipher.Block, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}
	if len(keyBytes) != 32 {
		return nil, errors.New("encryption key must be 32 bytes")
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// pkcs7Padding adds PKCS7 padding to the data
func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// pkcs7Unpadding removes PKCS7 padding from the data
func pkcs7Unpadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("input data is empty")
	}

	unpadding := int(data[length-1])
	if unpadding > length {
		return nil, errors.New("invalid padding")
	}

	return data[:length-unpadding], nil
}

// Encrypt performs double encryption on the plaintext using two keys in CBC mode.
func Encrypt(plaintext string) (string, error) {
	// First encryption with EncryptionKey1
	block1, err := getCipher(config.AppConfig.EncryptionKey1)
	if err != nil {
		return "", err
	}

	// Use the first key as IV for first encryption
	iv1, err := base64.StdEncoding.DecodeString(config.AppConfig.EncryptionKey1)
	if err != nil || len(iv1) < block1.BlockSize() {
		return "", errors.New("invalid IV for first encryption")
	}
	iv1 = iv1[:block1.BlockSize()]

	// Pad the plaintext
	paddedData := pkcs7Padding([]byte(plaintext), block1.BlockSize())

	// Create CBC encrypter
	cbc1 := cipher.NewCBCEncrypter(block1, iv1)
	ciphertext1 := make([]byte, len(paddedData))
	cbc1.CryptBlocks(ciphertext1, paddedData)

	// Base64 encode first encryption
	encrypted1B64 := base64.StdEncoding.EncodeToString(ciphertext1)

	// Second encryption with EncryptionKey2
	block2, err := getCipher(config.AppConfig.EncryptionKey2)
	if err != nil {
		return "", err
	}

	// Use the second key as IV for second encryption
	iv2, err := base64.StdEncoding.DecodeString(config.AppConfig.EncryptionKey2)
	if err != nil || len(iv2) < block2.BlockSize() {
		return "", errors.New("invalid IV for second encryption")
	}
	iv2 = iv2[:block2.BlockSize()]

	// Pad the first encrypted text
	paddedData2 := pkcs7Padding([]byte(encrypted1B64), block2.BlockSize())

	// Create CBC encrypter
	cbc2 := cipher.NewCBCEncrypter(block2, iv2)
	ciphertext2 := make([]byte, len(paddedData2))
	cbc2.CryptBlocks(ciphertext2, paddedData2)

	// Base64 encode final ciphertext
	encrypted2B64 := base64.StdEncoding.EncodeToString(ciphertext2)

	return encrypted2B64, nil
}

// Decrypt reverses the double encryption process to retrieve the plaintext.
func Decrypt(ciphertext string) (string, error) {
	// Base64 decode the outer ciphertext
	encrypted2, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	// First decryption with EncryptionKey2
	block2, err := getCipher(config.AppConfig.EncryptionKey2)
	if err != nil {
		return "", err
	}

	// Use the second key as IV for first decryption
	iv2, err := base64.StdEncoding.DecodeString(config.AppConfig.EncryptionKey2)
	if err != nil || len(iv2) < block2.BlockSize() {
		return "", errors.New("invalid IV for second decryption")
	}
	iv2 = iv2[:block2.BlockSize()]

	// Create CBC decrypter
	cbc2 := cipher.NewCBCDecrypter(block2, iv2)
	decrypted2 := make([]byte, len(encrypted2))
	cbc2.CryptBlocks(decrypted2, encrypted2)

	// Remove padding
	decrypted2, err = pkcs7Unpadding(decrypted2)
	if err != nil {
		return "", err
	}

	// Base64 decode the inner encrypted text
	encrypted1B64 := string(decrypted2)
	encrypted1, err := base64.StdEncoding.DecodeString(encrypted1B64)
	if err != nil {
		return "", err
	}

	// Second decryption with EncryptionKey1
	block1, err := getCipher(config.AppConfig.EncryptionKey1)
	if err != nil {
		return "", err
	}
	// Use the first key as IV for second decryption
	iv1, err := base64.StdEncoding.DecodeString(config.AppConfig.EncryptionKey1)
	if err != nil || len(iv1) < block1.BlockSize() {
		return "", errors.New("invalid IV for first decryption")
	}
	iv1 = iv1[:block1.BlockSize()]

	// Create CBC decrypter
	cbc1 := cipher.NewCBCDecrypter(block1, iv1)
	decrypted1 := make([]byte, len(encrypted1))
	cbc1.CryptBlocks(decrypted1, encrypted1)

	// Remove padding
	decrypted1, err = pkcs7Unpadding(decrypted1)
	if err != nil {
		return "", err
	}

	return string(decrypted1), nil
}
