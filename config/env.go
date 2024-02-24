package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MQTTServer         string
	EncryptionKey      string
	SonosClientId      string
	SonosClientSecret  string
	SonosHouseholdId   string
	HueAuthToken       string
	EncryptionFilePath string
}

var config Config

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func LoadEnvs() {
	config = Config{
		MQTTServer:         getEnvVarOrDefault("MQTT_SERVER", "tcp://localhost:1883"),
		EncryptionKey:      getEnvVarOrDefault("ENCRYPTION_KEY", "example key 1234"),
		SonosClientId:      getEnvVarOrDefault("SONOS_CLIENT_ID", ""),
		SonosClientSecret:  getEnvVarOrDefault("SONOS_CLIENT_SECRET", ""),
		SonosHouseholdId:   getEnvVarOrDefault("SONOS_HOUSEHOLD_ID", ""),
		HueAuthToken:       getEnvVarOrDefault("HUE_AUTH_TOKEN", ""),
		EncryptionFilePath: getEnvVarOrDefault("ENCRYPTION_FILE_PATH", "tokens"),
	}
}

func getEnvVarOrDefault(envVar string, defaultValue string) string {
	value, exists := os.LookupEnv(envVar)
	if exists {
		return value
	}
	return defaultValue
}

func Get() *Config {
	return &config
}
