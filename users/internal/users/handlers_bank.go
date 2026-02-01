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
)

func UserBankAccountHandler(storeDDB *dynamo.Store) http.HandlerFunc {
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

		idUser, ok := claims["sub"].(string)
		if !ok || idUser == "" {
			http.Error(w, "ID do usuario invalido", http.StatusUnauthorized)
			return
		}

		var req struct {
			Banco     string `json:"banco"`
			BancoNome string `json:"banco_nome"`
			Conta     string `json:"conta"`
			Agencia   string `json:"agencia"`
			Digito    string `json:"digito"`
			CPF       string `json:"cpf"`
			Telefone  string `json:"telefone"`
			Pix       string `json:"pix"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao processar o JSON", http.StatusBadRequest)
			return
		}

		if req.Banco == "" || req.BancoNome == "" || req.Conta == "" || req.Agencia == "" || req.Digito == "" || req.CPF == "" || req.Telefone == "" {
			http.Error(w, "Todos os campos sao obrigatorios", http.StatusBadRequest)
			return
		}

		id := uuid.NewString()
		now := time.Now().Format(time.RFC3339)

		bankItem := map[string]types.AttributeValue{
			"PK":          dynamo.S(store.UserPK(idUser)),
			"SK":          dynamo.S("BANK#" + id),
			"id":          dynamo.S(id),
			"id_user":     dynamo.S(idUser),
			"banco":       dynamo.S(req.Banco),
			"banco_nome":  dynamo.S(req.BancoNome),
			"conta":       dynamo.S(req.Conta),
			"agencia":     dynamo.S(req.Agencia),
			"digito":      dynamo.S(req.Digito),
			"cpf":         dynamo.S(req.CPF),
			"telefone":    dynamo.S(req.Telefone),
			"pix":         dynamo.S(req.Pix),
			"active":      dynamo.B(true),
			"dell":        dynamo.B(false),
			"date_create": dynamo.S(now),
			"date_update": dynamo.S(""),
		}

		lookupItem := map[string]types.AttributeValue{
			"PK":      dynamo.S(store.BankPK(id)),
			"SK":      dynamo.S(store.UserPK(idUser)),
			"id":      dynamo.S(id),
			"id_user": dynamo.S(idUser),
			"active":  dynamo.B(true),
			"dell":    dynamo.B(false),
		}

		ctx := r.Context()
		err = storeDDB.TransactWrite(ctx, []types.TransactWriteItem{
			{Put: &types.Put{TableName: &storeDDB.Table, Item: bankItem}},
			{Put: &types.Put{TableName: &storeDDB.Table, Item: lookupItem}},
		})
		if err != nil {
			http.Error(w, "Erro ao salvar os dados bancarios: "+err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusCreated, map[string]string{
			"message": "Conta bancaria cadastrada com sucesso",
			"id":      id,
		})
	}
}

func UserBankAccountUpdateHandler(storeDDB *dynamo.Store) http.HandlerFunc {
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

		idUser, ok := claims["sub"].(string)
		if !ok || idUser == "" {
			http.Error(w, "ID do usuario invalido", http.StatusUnauthorized)
			return
		}

		var req struct {
			IDContaOld string `json:"id_conta_old"`
			Banco      string `json:"banco"`
			BancoNome  string `json:"banco_nome"`
			Conta      string `json:"conta"`
			Agencia    string `json:"agencia"`
			Digito     string `json:"digito"`
			CPF        string `json:"cpf"`
			Telefone   string `json:"telefone"`
			Pix        string `json:"pix"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao processar o JSON", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		oldItem, err := storeDDB.GetItem(ctx, store.UserPK(idUser), "BANK#"+req.IDContaOld)
		if err != nil || len(oldItem) == 0 {
			http.Error(w, "Conta antiga nao encontrada ou nao pertence ao usuario", http.StatusForbidden)
			return
		}
		if b, ok := oldItem["active"].(*types.AttributeValueMemberBOOL); ok && !b.Value {
			http.Error(w, "Conta antiga nao encontrada ou nao pertence ao usuario", http.StatusForbidden)
			return
		}

		_ = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
			"PK": dynamo.S(store.UserPK(idUser)),
			"SK": dynamo.S("BANK#" + req.IDContaOld),
		}, "SET active = :a, dell = :d, date_update = :u", nil, map[string]types.AttributeValue{
			":a": dynamo.B(false),
			":d": dynamo.B(true),
			":u": dynamo.S(time.Now().Format(time.RFC3339)),
		})

		newID := uuid.NewString()
		now := time.Now().Format(time.RFC3339)
		newItem := map[string]types.AttributeValue{
			"PK":          dynamo.S(store.UserPK(idUser)),
			"SK":          dynamo.S("BANK#" + newID),
			"id":          dynamo.S(newID),
			"id_user":     dynamo.S(idUser),
			"banco":       dynamo.S(req.Banco),
			"banco_nome":  dynamo.S(req.BancoNome),
			"conta":       dynamo.S(req.Conta),
			"agencia":     dynamo.S(req.Agencia),
			"digito":      dynamo.S(req.Digito),
			"cpf":         dynamo.S(req.CPF),
			"telefone":    dynamo.S(req.Telefone),
			"pix":         dynamo.S(req.Pix),
			"active":      dynamo.B(true),
			"dell":        dynamo.B(false),
			"date_create": dynamo.S(now),
			"date_update": dynamo.S(""),
		}
		lookupItem := map[string]types.AttributeValue{
			"PK":      dynamo.S(store.BankPK(newID)),
			"SK":      dynamo.S(store.UserPK(idUser)),
			"id":      dynamo.S(newID),
			"id_user": dynamo.S(idUser),
			"active":  dynamo.B(true),
			"dell":    dynamo.B(false),
		}

		err = storeDDB.TransactWrite(ctx, []types.TransactWriteItem{
			{Put: &types.Put{TableName: &storeDDB.Table, Item: newItem}},
			{Put: &types.Put{TableName: &storeDDB.Table, Item: lookupItem}},
		})
		if err != nil {
			http.Error(w, "Erro ao criar nova conta: "+err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"message": "Conta atualizada com sucesso",
			"new_id":  newID,
			"old_id":  req.IDContaOld,
		})
	}
}

func UserBankAccountGetHandler(storeDDB *dynamo.Store) http.HandlerFunc {
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

		idFromQuery := r.URL.Query().Get("id_user")
		if idFromQuery == "" {
			http.Error(w, "Parametro id_user e obrigatorio", http.StatusBadRequest)
			return
		}

		if idFromToken != idFromQuery {
			http.Error(w, "Usuario nao autorizado a acessar esta conta bancaria", http.StatusForbidden)
			return
		}

		ctx := r.Context()
		out, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S(store.UserPK(idFromToken)),
				":sk": dynamo.S("BANK#"),
			},
		})
		if err != nil || len(out.Items) == 0 {
			http.Error(w, "Nenhuma conta ativa encontrada para este usuario", http.StatusNotFound)
			return
		}

		var conta map[string]interface{}
		for _, item := range out.Items {
			if b, ok := item["active"].(*types.AttributeValueMemberBOOL); ok && b.Value {
				if d, ok := item["dell"].(*types.AttributeValueMemberBOOL); ok && !d.Value {
					var m map[string]interface{}
					attributevalue.UnmarshalMap(item, &m)
					conta = m
					break
				}
			}
		}
		if conta == nil {
			http.Error(w, "Nenhuma conta ativa encontrada para este usuario", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         conta["id"],
			"banco":      conta["banco"],
			"banco_nome": conta["banco_nome"],
			"conta":      conta["conta"],
			"agencia":    conta["agencia"],
			"digito":     conta["digito"],
			"cpf":        conta["cpf"],
			"telefone":   conta["telefone"],
			"pix":        conta["pix"],
		})
	}
}
