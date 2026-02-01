package users

import (
	"BACK_SORTE_GO/internal/store"
	"BACK_SORTE_GO/internal/store/dynamo"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecretKey = []byte("SUA_CHAVE_SECRETA")

func CreateUserHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
			CPF      string `json:"cpf"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao decodificar o JSON", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Email == "" || req.Password == "" || req.CPF == "" {
			http.Error(w, "Todos os campos (name, email, password, cpf) sao obrigatorios", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		existsOut, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			IndexName:              aws.String("GSI2"),
			KeyConditionExpression: aws.String("GSI2PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S("EMAIL#" + strings.ToLower(req.Email)),
			},
			Limit: aws.Int32(1),
		})
		if err != nil {
			http.Error(w, "Erro ao verificar duplicacao de email: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if len(existsOut.Items) > 0 {
			http.Error(w, "O email ja esta em uso", http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Erro ao processar a senha", http.StatusInternalServerError)
			return
		}

		userID := uuid.NewString()
		now := time.Now().Format(time.RFC3339)

		userItem := map[string]types.AttributeValue{
			"PK":          dynamo.S(store.UserPK(userID)),
			"SK":          dynamo.S("PROFILE"),
			"GSI2PK":      dynamo.S("EMAIL#" + strings.ToLower(req.Email)),
			"GSI2SK":      dynamo.S(store.UserPK(userID)),
			"id":          dynamo.S(userID),
			"name":        dynamo.S(req.Name),
			"email":       dynamo.S(req.Email),
			"password":    dynamo.S(string(hashedPassword)),
			"cpf":         dynamo.S(req.CPF),
			"active":      dynamo.B(true),
			"inicial":     dynamo.B(false),
			"dell":        dynamo.B(false),
			"date_create": dynamo.S(now),
			"date_update": dynamo.S(""),
		}

		contaNivelID := uuid.NewString()
		contaNivelItem := map[string]types.AttributeValue{
			"PK":             dynamo.S(store.UserPK(userID)),
			"SK":             dynamo.S("ACCOUNT#LEVEL"),
			"id":             dynamo.S(contaNivelID),
			"id_user":        dynamo.S(userID),
			"nivel":          dynamo.S("BASICO"),
			"ativo":          dynamo.B(false),
			"status":         dynamo.S("INATIVO"),
			"data_pagamento": dynamo.S(""),
			"tipo_pagamento": dynamo.S("INATIVO"),
			"data_update":    dynamo.S(now),
		}

		contaNivelPagID := uuid.NewString()
		contaNivelPagItem := map[string]types.AttributeValue{
			"PK":            dynamo.S(store.UserPK(userID)),
			"SK":            dynamo.S("ACCOUNT#PAYMENT#" + contaNivelPagID),
			"id":            dynamo.S(contaNivelPagID),
			"id_user":       dynamo.S(userID),
			"pago_data":     dynamo.S(""),
			"pago":          dynamo.B(false),
			"valor":         dynamo.S("0"),
			"status":        dynamo.S("INATIVO"),
			"codigo":        dynamo.S("111"),
			"data_create":   dynamo.S(now),
			"referente":     dynamo.S("01"),
			"valido":        dynamo.B(true),
			"txid":          dynamo.S(""),
			"pg_status":     dynamo.S("INATIVO"),
			"cpf":           dynamo.S(""),
			"chave":         dynamo.S(""),
			"pixCopiaECola": dynamo.S(""),
			"expiracao":     dynamo.S(""),
		}

		err = storeDDB.TransactWrite(ctx, []types.TransactWriteItem{
			{Put: &types.Put{TableName: &storeDDB.Table, Item: userItem}},
			{Put: &types.Put{TableName: &storeDDB.Table, Item: contaNivelItem}},
			{Put: &types.Put{TableName: &storeDDB.Table, Item: contaNivelPagItem}},
		})
		if err != nil {
			http.Error(w, "Erro ao criar o usuario: "+err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusCreated, map[string]string{
			"message": "Usuario criado com sucesso",
			"id":      userID,
		})
	}
}

func UserShowHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		if id == "" {
			http.Error(w, "ID do usuario nao fornecido", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		item, err := storeDDB.GetItem(ctx, store.UserPK(id), "PROFILE")
		if err != nil || len(item) == 0 {
			http.Error(w, "Usuario nao encontrado", http.StatusNotFound)
			return
		}
		var u userItem
		if err := attributevalue.UnmarshalMap(item, &u); err != nil {
			http.Error(w, "Erro ao buscar usuario", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"name":        u.Name,
			"email":       u.Email,
			"date_create": u.DateCreate,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

type UserNameChangeRequest struct {
	IDUser  string `json:"id_user"`
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

func UserNameChangeHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token nao fornecido", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token invalido", http.StatusUnauthorized)
			return
		}

		idFromToken, ok := claims["sub"].(string)
		if !ok || idFromToken == "" {
			http.Error(w, "ID do usuario invalido", http.StatusUnauthorized)
			return
		}

		var req UserNameChangeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao processar o JSON", http.StatusBadRequest)
			return
		}

		if req.IDUser != idFromToken {
			http.Error(w, "Usuario nao autorizado a alterar este nome", http.StatusForbidden)
			return
		}

		ctx := r.Context()
		item, err := storeDDB.GetItem(ctx, store.UserPK(req.IDUser), "PROFILE")
		if err != nil || len(item) == 0 {
			http.Error(w, "Usuario nao encontrado", http.StatusNotFound)
			return
		}
		var u userItem
		if err := attributevalue.UnmarshalMap(item, &u); err != nil {
			http.Error(w, "Erro ao buscar usuario", http.StatusInternalServerError)
			return
		}
		if u.Name != req.OldName {
			http.Error(w, "O nome antigo nao corresponde ao cadastrado", http.StatusBadRequest)
			return
		}

		err = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
			"PK": dynamo.S(store.UserPK(req.IDUser)),
			"SK": dynamo.S("PROFILE"),
		}, "SET name = :n, date_update = :d", nil, map[string]types.AttributeValue{
			":n": dynamo.S(req.NewName),
			":d": dynamo.S(time.Now().Format(time.RFC3339)),
		})
		if err != nil {
			http.Error(w, "Erro ao atualizar nome", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Nome atualizado com sucesso",
		})
	}
}
