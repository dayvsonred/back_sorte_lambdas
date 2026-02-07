package router

import (
	"net/http"

	"BACK_SORTE_GO/internal/handlers"
	"BACK_SORTE_GO/internal/utils"

	"github.com/gorilla/mux"
)

func New(h *handlers.Handler) *mux.Router {
	r := mux.NewRouter()
	r.Use(utils.CorsMiddleware)
	r.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	r.HandleFunc("/payments/donations", h.CreateDonation).Methods(http.MethodPost)
	r.HandleFunc("/payments/intents", h.CreatePaymentIntent).Methods(http.MethodPost)
	return r
}
