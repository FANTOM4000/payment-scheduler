package config

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	PocketBase    PocketBaseConfig
	Imap          ImapConfig
	PaymentConfig PaymentConfig
}

type PocketBaseConfig struct {
	Address  string `envconfig:"POCKETBASE_URL"`
	Email    string `envconfig:"POCKETBASE_EMAIL"`
	Password string `envconfig:"POCKETBASE_PASSWORD"`
}

type ImapConfig struct {
	Server   string `envconfig:"IMAP_HOST"`
	Port     int    `envconfig:"IMAP_PORT"`
	Email    string `envconfig:"IMAP_EMAIL"`
	Password string `envconfig:"IMAP_PASSWORD"`
	Mailbox  string `envconfig:"IMAP_MAILBOX"`
}

type PaymentConfig struct {
	Email    string `envconfig:"PAYMENT_EMAIL"`
	Password string `envconfig:"PAYMENT_PASSWORD"`
}

func LoadConfig() Config {
	var cfg Config
	err := godotenv.Load()
	if err != nil {
		_ = godotenv.Load("../.env") // Try loading from parent directory
	}
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("read env error : %s", err.Error())
	}
	return cfg
}
