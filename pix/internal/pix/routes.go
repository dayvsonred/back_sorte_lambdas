package pix

import (
	"BACK_SORTE_GO/internal/app"
	"BACK_SORTE_GO/internal/donation"

	"github.com/gorilla/mux"
)

func RegisterRoutes(router *mux.Router, a *app.App) {
	router.HandleFunc("/pix/create", CreatePixTokenHandler(a.Store)).Methods("POST")
	router.HandleFunc("/pix/status/{txid}", PixChargeStatusHandler()).Methods("GET")
	router.HandleFunc("/pix/monitora/{txid}", MonitorarStatusPagamentoHandler(a.Store)).Methods("POST")
	router.HandleFunc("/pix/total/{id}", donation.DonationSummaryByIDHandler(a.Store)).Methods("GET")
	router.HandleFunc("/pix/monitora/all", MonitorarStatusAllPagamentosHandler(a.Store)).Methods("GET")
}
