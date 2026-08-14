package manifest

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// DecryptManifestPayload decrypts an AES-256 encrypted payload blob using a hex-encoded depot decryption key.
func DecryptManifestPayload(encryptedPayload []byte, hexKey string) ([]byte, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex decryption key: %w", err)
	}

	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("expected 32-byte AES key (64 hex characters), got %d bytes", len(keyBytes))
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	blockSize := block.BlockSize()
	if len(encryptedPayload) < blockSize || len(encryptedPayload)%blockSize != 0 {
		return nil, fmt.Errorf("invalid encrypted payload size %d", len(encryptedPayload))
	}

	// First block is IV for CBC mode, or ECB if IV is zeroes
	iv := encryptedPayload[:blockSize]
	ciphertext := encryptedPayload[blockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(ciphertext))
	mode.CryptBlocks(decrypted, ciphertext)

	// Remove PKCS7 padding if present
	if len(decrypted) > 0 {
		padding := int(decrypted[len(decrypted)-1])
		if padding > 0 && padding <= blockSize && padding <= len(decrypted) {
			validPadding := true
			for i := len(decrypted) - padding; i < len(decrypted); i++ {
				if int(decrypted[i]) != padding {
					validPadding = false
					break
				}
			}
			if validPadding {
				decrypted = decrypted[:len(decrypted)-padding]
			}
		}
	}

	return decrypted, nil
}

// DecryptFilename decrypts a base64-encoded encrypted filename using a 32-byte (64-character hex) depot decryption key.
func DecryptFilename(b64EncryptedName string, hexKey string) (string, error) {
	if hexKey == "" {
		return b64EncryptedName, nil
	}

	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil || len(keyBytes) != 32 {
		return b64EncryptedName, err
	}

	decodedData, err := base64.StdEncoding.DecodeString(b64EncryptedName)
	if err != nil || len(decodedData) < 16 {
		return b64EncryptedName, nil
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return b64EncryptedName, err
	}

	// First 16 bytes: ECB-encrypted IV
	iv := make([]byte, 16)
	block.Decrypt(iv, decodedData[:16])

	ciphertext := decodedData[16:]
	if len(ciphertext) == 0 || len(ciphertext)%16 != 0 {
		return b64EncryptedName, nil
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decryptedPadded := make([]byte, len(ciphertext))
	mode.CryptBlocks(decryptedPadded, ciphertext)

	// Remove PKCS7 padding
	if len(decryptedPadded) > 0 {
		padding := int(decryptedPadded[len(decryptedPadded)-1])
		if padding > 0 && padding <= 16 && padding <= len(decryptedPadded) {
			valid := true
			for i := len(decryptedPadded) - padding; i < len(decryptedPadded); i++ {
				if int(decryptedPadded[i]) != padding {
					valid = false
					break
				}
			}
			if valid {
				decryptedPadded = decryptedPadded[:len(decryptedPadded)-padding]
			}
		}
	}

	res := strings.TrimRight(string(decryptedPadded), "\x00")
	return res, nil
}

// DecryptChunkPayload decrypts an AES-256 encrypted chunk payload blob downloaded from a Steam CDN.
func DecryptChunkPayload(encryptedData []byte, hexKey string) ([]byte, error) {
	if hexKey == "" {
		return encryptedData, nil
	}

	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil || len(keyBytes) != 32 {
		return nil, fmt.Errorf("invalid 32-byte hex decryption key: %w", err)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	blockSize := block.BlockSize()
	if len(encryptedData) < blockSize {
		return nil, fmt.Errorf("chunk data length %d smaller than block size %d", len(encryptedData), blockSize)
	}

	// First block is IV for CBC mode
	iv := encryptedData[:blockSize]
	ciphertext := encryptedData[blockSize:]

	if len(ciphertext)%blockSize != 0 {
		// If not aligned to block size, return original (may be unencrypted)
		return encryptedData, nil
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(ciphertext))
	mode.CryptBlocks(decrypted, ciphertext)

	return decrypted, nil
}

