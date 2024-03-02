package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"cuore/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/oauth2"
)

var (
	authData = map[string]oauth2.Token{}
)

func GetTokenForProvider(provider string) (*oauth2.Token, error) {
	if token, ok := authData[provider]; ok {
		return &token, nil
	}

	token, err := LoadTokenFromFile(provider)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func SaveTokenForProvider(provider string, token *oauth2.Token) error {
	authData[provider] = *token

	err := SaveTokenToFile(provider, token)

	if err != nil {
		return err
	}

	return nil
}

func SaveTokenToFile(filename string, oauthToken *oauth2.Token) error {
	jsonToken, err := json.Marshal(oauthToken)
	if err != nil {
		return err
	}

	log.Print("Encrypting token")
	encryptedToken, err := encrypt(jsonToken)
	if err != nil {
		return err
	}

	log.Printf("Saving token to file: %s", filename)
	fullPath := fmt.Sprintf("%s/%s", config.Get().EncryptionFilePath, filename)
	err = os.WriteFile(fullPath, encryptedToken, 0644)
	if err != nil {
		return err
	}

	return nil
}

func LoadTokenFromFile(filename string) (*oauth2.Token, error) {
	log.Printf("Loading token from file: %s", filename)
	fullPath := fmt.Sprintf("%s/%s", config.Get().EncryptionFilePath, filename)
	encryptedToken, err := os.ReadFile(fullPath)
	if err != nil {
		log.Fatalf("Error reading token from file: %v", err)
		return nil, err
	}

	decryptedToken, err := decrypt(encryptedToken)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	err = json.Unmarshal(decryptedToken, &token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

// encrypt encrypts data using AES encryption.
func encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(config.Get().EncryptionKey))
	if err != nil {
		return nil, err
	}

	cipherText := make([]byte, aes.BlockSize+len(data))
	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], data)

	return cipherText, nil
}

// decrypt decrypts data using AES decryption.
func decrypt(cipherText []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(config.Get().EncryptionKey))
	if err != nil {
		return nil, err
	}

	if len(cipherText) < aes.BlockSize {
		return nil, fmt.Errorf("cipherText too short")
	}
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	return cipherText, nil
}
