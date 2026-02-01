package login

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"BACK_SORTE_GO/internal/store"
	"BACK_SORTE_GO/internal/store/dynamo"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecretKey = []byte("SUA_CHAVE_SECRETA")

type ContaNivel struct {
	ID            string     `json:"id"`
	IDUser        string     `json:"id_user"`
	Nivel         string     `json:"nivel"`
	Ativo         bool       `json:"ativo"`
	Status        string     `json:"status"`
	DataPagamento *time.Time `json:"data_pagamento,omitempty"`
	TipoPagamento string     `json:"tipo_pagamento"`
	DataUpdate    time.Time  `json:"data_update"`
}

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

type contaNivelItem struct {
	ID            string `dynamodbav:"id"`
	IDUser        string `dynamodbav:"id_user"`
	Nivel         string `dynamodbav:"nivel"`
	Ativo         bool   `dynamodbav:"ativo"`
	Status        string `dynamodbav:"status"`
	DataPagamento string `dynamodbav:"data_pagamento"`
	TipoPagamento string `dynamodbav:"tipo_pagamento"`
	DataUpdate    string `dynamodbav:"data_update"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID         string    `json:"id"`
		Name       string    `json:"name"`
		Email      string    `json:"email"`
		CPF        string    `json:"cpf"`
		Active     bool      `json:"active"`
		Inicial    bool      `json:"inicial"`
		Dell       bool      `json:"dell"`
		DateCreate time.Time `json:"date_create"`
	} `json:"user"`
	ContaNivel *ContaNivel `json:"conta_nivel,omitempty"`
}

func LoginHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		contentType := r.Header.Get("Content-Type")
		if authHeader != "Basic QVBJX05BTUVfQUNDRVNTOkFQSV9TRUNSRVRfQUNDRVNT" || contentType != "application/x-www-form-urlencoded" {
			http.Error(w, "Cabecalhos invalidos", http.StatusUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Erro ao processar os parametros", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")
		grantType := r.FormValue("grant_type")

		if grantType != "password" || username == "" || password == "" {
			http.Error(w, "Parametros invalidos", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		out, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			IndexName:              aws.String("GSI2"),
			KeyConditionExpression: aws.String("GSI2PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S("EMAIL#" + strings.ToLower(username)),
			},
			Limit: aws.Int32(1),
		})
		if err != nil || len(out.Items) == 0 {
			http.Error(w, "Usuario ou senha invalidos", http.StatusUnauthorized)
			return
		}

		var user userItem
		if err := attributevalue.UnmarshalMap(out.Items[0], &user); err != nil {
			http.Error(w, "Erro ao buscar usuario", http.StatusInternalServerError)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
			http.Error(w, "Usuario ou senha invalidos", http.StatusUnauthorized)
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": user.ID,
			"exp": time.Now().Add(time.Hour * 24).Unix(),
		})
		tokenString, err := token.SignedString(jwtSecretKey)
		if err != nil {
			http.Error(w, "Erro ao gerar token", http.StatusInternalServerError)
			return
		}

		var contaNivel *ContaNivel
		if item, err := storeDDB.GetItem(ctx, store.UserPK(user.ID), "ACCOUNT#LEVEL"); err == nil && len(item) > 0 {
			var raw contaNivelItem
			if err := attributevalue.UnmarshalMap(item, &raw); err == nil {
				var dtUpdate time.Time
				if raw.DataUpdate != "" {
					dtUpdate, _ = time.Parse(time.RFC3339, raw.DataUpdate)
				}
				var dtPag *time.Time
				if raw.DataPagamento != "" {
					if t, err := time.Parse(time.RFC3339, raw.DataPagamento); err == nil {
						dtPag = &t
					}
				}
				contaNivel = &ContaNivel{
					ID:            raw.ID,
					IDUser:        raw.IDUser,
					Nivel:         raw.Nivel,
					Ativo:         raw.Ativo,
					Status:        raw.Status,
					DataPagamento: dtPag,
					TipoPagamento: raw.TipoPagamento,
					DataUpdate:    dtUpdate,
				}
			}
		}

		response := LoginResponse{
			Token: tokenString,
			User: struct {
				ID         string    `json:"id"`
				Name       string    `json:"name"`
				Email      string    `json:"email"`
				CPF        string    `json:"cpf"`
				Active     bool      `json:"active"`
				Inicial    bool      `json:"inicial"`
				Dell       bool      `json:"dell"`
				DateCreate time.Time `json:"date_create"`
			}{
				ID:         user.ID,
				Name:       user.Name,
				Email:      user.Email,
				CPF:        user.CPF,
				Active:     user.Active,
				Inicial:    user.Inicial,
				Dell:       user.Dell,
				DateCreate: parseTimeOrNow(user.DateCreate),
			},
		}

		if contaNivel != nil {
			response.ContaNivel = contaNivel
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
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

