package login

import (
	"BACK_SORTE_GO/internal/app"

	"github.com/gorilla/mux"
)

func RegisterRoutes(router *mux.Router, a *app.App) {
	router.HandleFunc("/login", LoginHandler(a.Store)).Methods("POST")
}
