package contact

import (
	"BACK_SORTE_GO/internal/app"

	"github.com/gorilla/mux"
)

func RegisterRoutes(router *mux.Router, a *app.App) {
	router.HandleFunc("/contact/health", ContactHealthHandler()).Methods("GET")
	router.HandleFunc("/contact/mensagem", ContactMensagemHandler(a.Store)).Methods("POST")
	router.HandleFunc("/contact/visualizations", ContactVisualizationHandler(a.Store)).Methods("POST")
}
