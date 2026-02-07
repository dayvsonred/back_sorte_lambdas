package router

import (
	"net/http"

	"BACK_SORTE_GO/internal/handlers"
	"BACK_SORTE_GO/internal/utils"

	"github.com/gorilla/mux"
)

func New(h *handlers.Handler) *mux.Router {
	router := mux.NewRouter()
	router.Use(utils.CorsMiddleware)
	router.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	router.HandleFunc("/payments/health", h.Health(router)).Methods(http.MethodGet)
	router.HandleFunc("/payments/donations", h.CreateDonation).Methods(http.MethodPost)
	router.HandleFunc("/payments/intents", h.CreatePaymentIntent).Methods(http.MethodPost)
	router.HandleFunc("/payments/checkout-session", h.CreateCheckoutSession).Methods(http.MethodPost)
	return router
}
