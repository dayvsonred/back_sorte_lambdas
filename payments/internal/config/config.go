package config

import (
	"errors"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	StripeSecretKey string
	DynamoTableName string
	AwsRegion       string
	Env             string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		StripeSecretKey: strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY")),
		DynamoTableName: strings.TrimSpace(os.Getenv("DYNAMO_TABLE_NAME")),
		AwsRegion:       strings.TrimSpace(os.Getenv("AWS_REGION")),
		Env:             strings.TrimSpace(os.Getenv("ENV")),
	}

	if cfg.Env == "" {
		cfg.Env = "dev"
	}

	if cfg.StripeSecretKey == "" {
		return Config{}, errors.New("STRIPE_SECRET_KEY nao definido")
	}
	if cfg.DynamoTableName == "" {
		return Config{}, errors.New("DYNAMO_TABLE_NAME nao definido")
	}
	if cfg.AwsRegion == "" {
		return Config{}, errors.New("AWS_REGION nao definido")
	}

	return cfg, nil
}
