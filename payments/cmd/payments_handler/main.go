package main

import (
	"context"
	"encoding/json"
	"log"

	"BACK_SORTE_GO/internal/config"
	"BACK_SORTE_GO/internal/dynamo"
	"BACK_SORTE_GO/internal/handlers"
	"BACK_SORTE_GO/internal/router"
	"BACK_SORTE_GO/internal/stripeclient"
	"BACK_SORTE_GO/internal/utils"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Erro ao carregar config: %v", err)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AwsRegion))
	if err != nil {
		log.Fatalf("Erro ao carregar config AWS: %v", err)
	}

	ddb := dynamodb.NewFromConfig(awsCfg)
	store := dynamo.New(ddb, cfg.DynamoTableName)
	stripeClient := stripeclient.New(cfg.StripeSecretKey)

	logger := utils.NewLogger()
	h := handlers.NewHandler(store, stripeClient, cfg, logger)
	muxRouter := router.New(h)

	adapter := httpadapter.NewV2(muxRouter)

	handler := func(ctx context.Context, raw json.RawMessage) (interface{}, error) {
		var apiEvent events.APIGatewayV2HTTPRequest
		if err := json.Unmarshal(raw, &apiEvent); err == nil {
			if apiEvent.RequestContext.HTTP.Method != "" {
				return adapter.ProxyWithContext(ctx, apiEvent)
			}
		}

		var ebEvent events.EventBridgeEvent
		if err := json.Unmarshal(raw, &ebEvent); err == nil {
			if len(ebEvent.Detail) > 0 {
				return h.HandleEventBridge(ctx, ebEvent)
			}
		}

		logger.Error("evento_nao_reconhecido", map[string]interface{}{})
		return map[string]string{"status": "ignored"}, nil
	}

	lambda.Start(handler)
}
