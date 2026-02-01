package users

import (
	"encoding/json"
	"net/http"
	"time"
)

type userItem struct {
	ID         string `dynamodbav:"id"`
	Name       string `dynamodbav:"name"`
	Email      string `dynamodbav:"email"`
	Password   string `dynamodbav:"password"`
	CPF        string `dynamodbav:"cpf"`
	Active     bool   `dynamodbav:"active"`
	Inicial    bool   `dynamodbav:"inicial"`
	Dell       bool   `dynamodbav:"dell"`
	DateCreate string `dynamodbav:"date_create"`
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func parseTimeOrNow(value string) time.Time {
	if value == "" {
		return time.Now()
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t
	}
	return time.Now()
}
