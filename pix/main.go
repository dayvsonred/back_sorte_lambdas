package main

import (
	"context"
	"log"

	"BACK_SORTE_GO/internal/app"
	"BACK_SORTE_GO/internal/pix"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/gorilla/mux"
)

func main() {
	ctx := context.Background()
	a, err := app.New(ctx)
	if err != nil {
		log.Fatalf("Erro ao inicializar app: %v", err)
	}

	router := mux.NewRouter()
	pix.RegisterRoutes(router, a)

	adapter := httpadapter.NewV2(router)
	lambda.Start(adapter.ProxyWithContext)
}
