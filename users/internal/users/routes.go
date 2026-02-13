package users

import (
	"BACK_SORTE_GO/internal/app"

	"github.com/gorilla/mux"
)

func RegisterRoutes(router *mux.Router, a *app.App) {
	router.HandleFunc("/users", CreateUserHandler(a.Store)).Methods("POST")
	router.HandleFunc("/users/passwordChange", UserPasswordChangeHandler(a.Store)).Methods("POST")
	router.HandleFunc("/users/passwordRecover", UserPasswordRecoverStartHandler(a.Store)).Methods("POST")
	router.HandleFunc("/users/passwordConfirmToken", UserPasswordRecoverConfirmHandler(a.Store)).Methods("POST")
	router.HandleFunc("/users/passwordRecoverLink", UserPasswordRecoverLinkHandler(a.Store)).Methods("GET")
	router.HandleFunc("/users/confirmEmail", UserConfirmEmailHandler(a.Store)).Methods("GET")
	router.HandleFunc("/users/bankAccount", UserBankAccountHandler(a.Store)).Methods("POST")
	router.HandleFunc("/users/bankAccount", UserBankAccountUpdateHandler(a.Store)).Methods("PATCH")
	router.HandleFunc("/users/bankAccount", UserBankAccountGetHandler(a.Store)).Methods("GET")
	router.HandleFunc("/users/uploadProfileImage", UploadUserProfileImageHandler(a.Store)).Methods("POST")
	router.HandleFunc("/users/ProfileImage/{id}", UserProfileImageHandler(a.Store)).Methods("GET")
	router.HandleFunc("/users/show/{id}", UserShowHandler(a.Store)).Methods("GET")
	router.HandleFunc("/users/nameChange", UserNameChangeHandler(a.Store)).Methods("POST")
}
