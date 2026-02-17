package admin

import (
	"BACK_SORTE_GO/internal/app"

	"github.com/gorilla/mux"
)

func RegisterRoutes(router *mux.Router, a *app.App) {
	router.HandleFunc("/admin/user/{id}", DeleteUserHandler(a.Store)).Methods("DELETE")
	router.HandleFunc("/admin/donation/{id}", DeleteDonationHandler(a.Store)).Methods("DELETE")
	router.HandleFunc("/admin/user/{id}/block", BlockUserHandler(a.Store)).Methods("POST")
}
