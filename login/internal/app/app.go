package app

import (
	"context"
	"fmt"

	"BACK_SORTE_GO/config"
	"BACK_SORTE_GO/internal/store/dynamo"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type App struct {
	Store *dynamo.Store
}

func New(ctx context.Context) (*App, error) {
	config.LoadEnv()

	region := config.GetAwsRegion()
	if region == "" {
		return nil, fmt.Errorf("AWS_REGION nao definido")
	}

	table := config.GetDynamoTableName()
	if table == "" {
		return nil, fmt.Errorf("DYNAMODB_TABLE nao definido")
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar config AWS: %w", err)
	}

	ddb := dynamodb.NewFromConfig(cfg)
	store := dynamo.New(ddb, table)

	return &App{Store: store}, nil
}
