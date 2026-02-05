package main

import (
	"context"
	"log"
	"net/http"

	"BACK_SORTE_GO/internal/app"
	"BACK_SORTE_GO/internal/middleware"
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
	router.Use(middleware.CorsMiddleware)
	router.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	pix.RegisterRoutes(router, a)

	adapter := httpadapter.NewV2(router)
	lambda.Start(adapter.ProxyWithContext)
}
