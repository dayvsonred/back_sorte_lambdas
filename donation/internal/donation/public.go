package donation

import (
	"BACK_SORTE_GO/internal/store"
	"BACK_SORTE_GO/internal/store/dynamo"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
)

type DonationMessageFull struct {
	ID          string    `json:"id"`
	Valor       string    `json:"valor"`
	CPF         string    `json:"cpf"`
	Nome        string    `json:"nome"`
	Mensagem    string    `json:"mensagem"`
	Anonimo     bool      `json:"anonimo"`
	DataCriacao time.Time `json:"data_criacao"`
}

type DonationSummary struct {
	ValorTotal    string `json:"valor_total"`
	TotalDoadores int    `json:"total_doadores"`
}

func DonationByLinkHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		nomeLink := vars["nome_link"]

		if nomeLink == "" || !strings.HasPrefix(nomeLink, "@") {
			http.Error(w, "nome_link invalido", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		out, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			KeyConditionExpression: aws.String("PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S(store.LinkPK(nomeLink)),
			},
			Limit: aws.Int32(1),
		})
		if err != nil || len(out.Items) == 0 {
			http.Error(w, "Doacao nao encontrada", http.StatusNotFound)
			return
		}
		linkItem := out.Items[0]
		idDoacao := ""
		if v, ok := linkItem["id_doacao"].(*types.AttributeValueMemberS); ok {
			idDoacao = v.Value
		}
		if idDoacao == "" {
			http.Error(w, "Doacao nao encontrada", http.StatusNotFound)
			return
		}

		profile, err := storeDDB.GetItem(ctx, store.DonationPK(idDoacao), "PROFILE")
		if err != nil || len(profile) == 0 {
			http.Error(w, "Erro ao buscar doacao", http.StatusInternalServerError)
			return
		}

		closed := false
		if v, ok := profile["closed"].(*types.AttributeValueMemberBOOL); ok {
			closed = v.Value
		}
		idUser := ""
		if v, ok := profile["id_user"].(*types.AttributeValueMemberS); ok {
			idUser = v.Value
		}

		if closed {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Doacao fechada. Acesso nao autorizado", http.StatusUnauthorized)
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

			idFromToken, ok := claims["sub"].(string)
			if !ok || idFromToken == "" || idFromToken != idUser {
				http.Error(w, "Voce nao tem permissao para acessar esta doacao fechada", http.StatusForbidden)
				return
			}
		}

		details, err := storeDDB.GetItem(ctx, store.DonationPK(idDoacao), "DETAILS")
		if err != nil || len(details) == 0 {
			http.Error(w, "Erro ao buscar detalhes", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{}
		attributevalue.UnmarshalMap(profile, &response)
		var dd map[string]interface{}
		attributevalue.UnmarshalMap(details, &dd)
		response["texto"] = dd["texto"]
		response["img_caminho"] = dd["img_caminho"]
		response["area"] = dd["area"]
		response["nome_link"] = nomeLink

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func DonationMensagesHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idDoacao := r.URL.Query().Get("id")
		if idDoacao == "" {
			http.Error(w, "Parametro 'id' e obrigatorio", http.StatusBadRequest)
			return
		}

		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")

		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			limit = 10
		}

		offset := (page - 1) * limit

		ctx := r.Context()
		out, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S(store.DonationPK(idDoacao)),
				":sk": dynamo.S(store.PrefixPix),
			},
			ScanIndexForward: aws.Bool(false),
		})
		if err != nil {
			http.Error(w, "Erro ao buscar mensagens: "+err.Error(), http.StatusInternalServerError)
			return
		}

		visible := make([]DonationMessageFull, 0)
		for _, item := range out.Items {
			if b, ok := item["visivel"].(*types.AttributeValueMemberBOOL); ok && b.Value {
				var msg DonationMessageFull
				attributevalue.UnmarshalMap(item, &msg)
				visible = append(visible, msg)
			}
		}

		end := offset + limit
		if offset > len(visible) {
			offset = len(visible)
		}
		if end > len(visible) {
			end = len(visible)
		}
		mensagens := visible[offset:end]

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mensagens)
	}
}

func DonationSummaryByIDHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idDoacao := vars["id"]
		if idDoacao == "" {
			http.Error(w, "Parametro 'id' e obrigatorio", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		out, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S(store.DonationPK(idDoacao)),
				":sk": dynamo.S(store.PrefixPix),
			},
		})
		if err != nil {
			http.Error(w, "Erro ao buscar resumo da doacao: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var total float64
		donors := map[string]struct{}{}
		for _, item := range out.Items {
			if b, ok := item["visivel"].(*types.AttributeValueMemberBOOL); ok && b.Value {
				if v, ok := item["valor"].(*types.AttributeValueMemberN); ok {
					val, _ := strconv.ParseFloat(v.Value, 64)
					total += val
				}
				if cpf, ok := item["cpf"].(*types.AttributeValueMemberS); ok {
					donors[cpf.Value] = struct{}{}
				}
			}
		}

		resumo := DonationSummary{
			ValorTotal:    fmt.Sprintf("%.2f", total),
			TotalDoadores: len(donors),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resumo)
	}
}
