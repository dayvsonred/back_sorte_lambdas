package donation

import (
	"BACK_SORTE_GO/internal/app"

	"github.com/gorilla/mux"
)

func RegisterRoutes(router *mux.Router, a *app.App) {
	router.HandleFunc("/donation", DonationHandler(a.Store)).Methods("POST")
	router.HandleFunc("/donation/list", DonationListByIDUserHandler(a.Store)).Methods("GET")
	router.HandleFunc("/donation/{id}", DonationDellHandler(a.Store)).Methods("DELETE")
	router.HandleFunc("/donation/link/{nome_link}", DonationByLinkHandler(a.Store)).Methods("GET")
	router.HandleFunc("/donation/mensagem", DonationMensagesHandler(a.Store)).Methods("GET")
	router.HandleFunc("/donation/closed/{id}", DonationClosedHandler(a.Store)).Methods("GET")
	router.HandleFunc("/donation/rescue/{id}", DonationRescueHandler(a.Store)).Methods("GET")
	router.HandleFunc("/donation/visualization", DonationVisualization(a.Store)).Methods("POST")
	router.HandleFunc("/donation/createUserAndDonation", DonationCreateSimpleHandler(a.Store)).Methods("POST")
}
