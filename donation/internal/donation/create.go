package donation

import (
	"BACK_SORTE_GO/config"
	"BACK_SORTE_GO/internal/store"
	"BACK_SORTE_GO/internal/store/dynamo"
	"BACK_SORTE_GO/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func DonationHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token nao fornecido", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey1, nil
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

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Erro ao ler formulario", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		valorStr := r.FormValue("valor")
		texto := r.FormValue("texto")
		area := r.FormValue("area")

		if idUser == "" || name == "" || valorStr == "" || texto == "" || area == "" {
			http.Error(w, "Todos os campos sao obrigatorios", http.StatusBadRequest)
			return
		}

		valor, err := strconv.ParseFloat(valorStr, 64)
		if err != nil || valor <= 0 {
			http.Error(w, "Valor invalido", http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "Imagem obrigatoria", http.StatusBadRequest)
			return
		}
		defer file.Close()

		imgFileName := fmt.Sprintf("%s_%d_%s", idUser, time.Now().Unix(), handler.Filename)
		imgPath, err := utils.UploadToS3(file, imgFileName, config.GetawsBucketNameImgDoacao())
		if err != nil {
			http.Error(w, "Erro ao subir imagem: "+err.Error(), http.StatusInternalServerError)
			return
		}

		donationID := uuid.NewString()
		now := time.Now().Format(time.RFC3339)
		nomeLink, err := generateUniqueLinkName(storeDDB, name)
		if err != nil {
			http.Error(w, "Erro ao gerar nome_link: "+err.Error(), http.StatusInternalServerError)
			return
		}

		donationItem := map[string]types.AttributeValue{
			"PK":          dynamo.S(store.DonationPK(donationID)),
			"SK":          dynamo.S("PROFILE"),
			"GSI1PK":      dynamo.S(store.UserPK(idUser)),
			"GSI1SK":      dynamo.S("DONATION#" + now + "#" + donationID),
			"id":          dynamo.S(donationID),
			"id_user":     dynamo.S(idUser),
			"name":        dynamo.S(name),
			"valor":       dynamo.N(fmt.Sprintf("%.2f", valor)),
			"active":      dynamo.B(true),
			"dell":        dynamo.B(false),
			"closed":      dynamo.B(false),
			"date_start":  dynamo.S(now),
			"date_create": dynamo.S(now),
			"date_update": dynamo.S(""),
			"nome_link":   dynamo.S(nomeLink),
		}

		detailsItem := map[string]types.AttributeValue{
			"PK":          dynamo.S(store.DonationPK(donationID)),
			"SK":          dynamo.S("DETAILS"),
			"id":          dynamo.S(uuid.NewString()),
			"id_doacao":   dynamo.S(donationID),
			"texto":       dynamo.S(texto),
			"img_caminho": dynamo.S(imgPath),
			"area":        dynamo.S(area),
		}

		linkItem := map[string]types.AttributeValue{
			"PK":        dynamo.S(store.LinkPK(nomeLink)),
			"SK":        dynamo.S("DONATION#" + donationID),
			"id":        dynamo.S(uuid.NewString()),
			"id_doacao": dynamo.S(donationID),
			"nome_link": dynamo.S(nomeLink),
			"id_user":   dynamo.S(idUser),
		}

		paymentItem := map[string]types.AttributeValue{
			"PK":               dynamo.S(store.DonationPK(donationID)),
			"SK":               dynamo.S("PAYMENT"),
			"id":               dynamo.S(uuid.NewString()),
			"id_doacao":        dynamo.S(donationID),
			"valor_disponivel": dynamo.N("0"),
			"valor_tranferido": dynamo.N("0"),
			"solicitado":       dynamo.B(false),
			"status":           dynamo.S("START"),
			"data_update":      dynamo.S(now),
		}

		ctx := r.Context()
		err = storeDDB.TransactWrite(ctx, []types.TransactWriteItem{
			{Put: &types.Put{TableName: &storeDDB.Table, Item: donationItem}},
			{Put: &types.Put{TableName: &storeDDB.Table, Item: detailsItem}},
			{Put: &types.Put{TableName: &storeDDB.Table, Item: linkItem}},
			{Put: &types.Put{TableName: &storeDDB.Table, Item: paymentItem}},
		})
		if err != nil {
			http.Error(w, "Erro ao salvar doacao: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message":   "Doacao criada com sucesso",
			"id":        donationID,
			"nome_link": nomeLink,
			"img":       imgPath,
		})
	}
}

func DonationCreateSimpleHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Erro ao ler formulario: "+err.Error(), http.StatusBadRequest)
			return
		}

		fullName := r.FormValue("fullName")
		cpf := r.FormValue("cpf")
		email := r.FormValue("email")
		senha := r.FormValue("senha")

		titulo := r.FormValue("titulo")
		metaStr := r.FormValue("meta")
		categoria := r.FormValue("categoria")
		texto := r.FormValue("texto")

		if fullName == "" || cpf == "" || email == "" || senha == "" || titulo == "" || metaStr == "" || categoria == "" || texto == "" {
			http.Error(w, "Campos obrigatorios ausentes", http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Erro ao gerar hash da senha: "+err.Error(), http.StatusInternalServerError)
			return
		}

		ctx := r.Context()
		existsOut, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			IndexName:              aws.String("GSI2"),
			KeyConditionExpression: aws.String("GSI2PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S("EMAIL#" + strings.ToLower(email)),
			},
			Limit: aws.Int32(1),
		})
		if err != nil {
			http.Error(w, "Erro ao verificar email: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if len(existsOut.Items) > 0 {
			http.Error(w, "Email ja cadastrado", http.StatusBadRequest)
			return
		}

		userID := uuid.NewString()
		now := time.Now().Format(time.RFC3339)

		userItem := map[string]types.AttributeValue{
			"PK":          dynamo.S(store.UserPK(userID)),
			"SK":          dynamo.S("PROFILE"),
			"GSI2PK":      dynamo.S("EMAIL#" + strings.ToLower(email)),
			"GSI2SK":      dynamo.S(store.UserPK(userID)),
			"id":          dynamo.S(userID),
			"name":        dynamo.S(fullName),
			"email":       dynamo.S(email),
			"password":    dynamo.S(string(hashedPassword)),
			"cpf":         dynamo.S(cpf),
			"active":      dynamo.B(true),
			"inicial":     dynamo.B(false),
			"dell":        dynamo.B(false),
			"date_create": dynamo.S(now),
			"date_update": dynamo.S(now),
		}

		contaNivelItem := map[string]types.AttributeValue{
			"PK":             dynamo.S(store.UserPK(userID)),
			"SK":             dynamo.S("ACCOUNT#LEVEL"),
			"id":             dynamo.S(uuid.NewString()),
			"id_user":        dynamo.S(userID),
			"nivel":          dynamo.S("BASICO"),
			"ativo":          dynamo.B(false),
			"status":         dynamo.S("INATIVO"),
			"tipo_pagamento": dynamo.S("INATIVO"),
			"data_update":    dynamo.S(now),
		}

		meta, err := utils.StringToFloat(metaStr)
		if err != nil {
			http.Error(w, "Meta invalida", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "Erro ao obter imagem: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		imgFileName := fmt.Sprintf("%s_%d_%s", userID, time.Now().Unix(), header.Filename)
		imgPath, err := utils.UploadToS3(file, imgFileName, config.GetawsBucketNameImgDoacao())
		if err != nil {
			http.Error(w, "Erro ao subir imagem: "+err.Error(), http.StatusInternalServerError)
			return
		}

		donationID := uuid.NewString()
		nomeLink, err := generateUniqueLinkName(storeDDB, titulo)
		if err != nil {
			http.Error(w, "Erro ao gerar nome_link: "+err.Error(), http.StatusInternalServerError)
			return
		}

		donationItem := map[string]types.AttributeValue{
			"PK":          dynamo.S(store.DonationPK(donationID)),
			"SK":          dynamo.S("PROFILE"),
			"GSI1PK":      dynamo.S(store.UserPK(userID)),
			"GSI1SK":      dynamo.S("DONATION#" + now + "#" + donationID),
			"id":          dynamo.S(donationID),
			"id_user":     dynamo.S(userID),
			"name":        dynamo.S(titulo),
			"valor":       dynamo.N(fmt.Sprintf("%.2f", meta)),
			"active":      dynamo.B(true),
			"dell":        dynamo.B(false),
			"closed":      dynamo.B(false),
			"date_start":  dynamo.S(now),
			"date_create": dynamo.S(now),
			"date_update": dynamo.S(now),
			"nome_link":   dynamo.S(nomeLink),
		}

		detailsItem := map[string]types.AttributeValue{
			"PK":          dynamo.S(store.DonationPK(donationID)),
			"SK":          dynamo.S("DETAILS"),
			"id":          dynamo.S(uuid.NewString()),
			"id_doacao":   dynamo.S(donationID),
			"texto":       dynamo.S(texto),
			"img_caminho": dynamo.S(imgPath),
			"area":        dynamo.S(categoria),
		}

		linkItem := map[string]types.AttributeValue{
			"PK":        dynamo.S(store.LinkPK(nomeLink)),
			"SK":        dynamo.S("DONATION#" + donationID),
			"id":        dynamo.S(uuid.NewString()),
			"id_doacao": dynamo.S(donationID),
			"nome_link": dynamo.S(nomeLink),
			"id_user":   dynamo.S(userID),
		}

		paymentItem := map[string]types.AttributeValue{
			"PK":               dynamo.S(store.DonationPK(donationID)),
			"SK":               dynamo.S("PAYMENT"),
			"id":               dynamo.S(uuid.NewString()),
			"id_doacao":        dynamo.S(donationID),
			"valor_disponivel": dynamo.N("0"),
			"valor_tranferido": dynamo.N("0"),
			"solicitado":       dynamo.B(false),
			"status":           dynamo.S("START"),
			"data_update":      dynamo.S(now),
		}

		err = storeDDB.TransactWrite(ctx, []types.TransactWriteItem{
			{Put: &types.Put{TableName: &storeDDB.Table, Item: userItem}},
			{Put: &types.Put{TableName: &storeDDB.Table, Item: contaNivelItem}},
			{Put: &types.Put{TableName: &storeDDB.Table, Item: donationItem}},
			{Put: &types.Put{TableName: &storeDDB.Table, Item: detailsItem}},
			{Put: &types.Put{TableName: &storeDDB.Table, Item: linkItem}},
			{Put: &types.Put{TableName: &storeDDB.Table, Item: paymentItem}},
		})
		if err != nil {
			http.Error(w, "Erro ao salvar dados: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "Usuario e doacao criados com sucesso",
			"user_id":   userID,
			"donation":  donationID,
			"nome_link": nomeLink,
			"img":       imgPath,
		})
	}
}
