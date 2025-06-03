package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

const (
	saltSize      = 16
	keySize       = 32
	applicationID = "com.teamwork-logger"
)

func Encrypt(plaintext string) (string, error) {
	key, err := deriveKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(encryptedText string) (string, error) {
	if encryptedText == "" {
		return "", nil
	}

	key, err := deriveKey()
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return encryptedText, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aesGCM.NonceSize() {
		return encryptedText, nil
	}

	nonce, ciphertext := ciphertext[:aesGCM.NonceSize()], ciphertext[aesGCM.NonceSize():]

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return encryptedText, nil
	}

	return string(plaintext), nil
}

func deriveKey() ([]byte, error) {
	machineID, err := getMachineID()
	if err != nil {
		return nil, err
	}

	combinedID := machineID + applicationID

	hash := sha256.Sum256([]byte(combinedID))
	return hash[:], nil
}

func getMachineID() (string, error) {
	var machineIDFile string

	switch runtime.GOOS {
	case "windows":
		machineIDFile = filepath.Join(os.Getenv("SYSTEMROOT"), "system32", "config", "systemprofile")
	case "darwin":
		machineIDFile = "/Library/Preferences/SystemConfiguration/com.apple.airport.preferences.plist"
	default:
		machineIDFile = "/etc/machine-id"
		if _, err := os.Stat(machineIDFile); os.IsNotExist(err) {
			machineIDFile = "/var/lib/dbus/machine-id"
		}
	}

	h := sha256.New()
	h.Write([]byte(machineIDFile))
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
