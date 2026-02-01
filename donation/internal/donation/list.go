package donation

import (
	"BACK_SORTE_GO/internal/store"
	"BACK_SORTE_GO/internal/store/dynamo"
	"encoding/json"
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

func DonationListByIDUserHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idUser := r.URL.Query().Get("id_user")
		if idUser == "" {
			http.Error(w, "Parametro 'id_user' e obrigatorio", http.StatusBadRequest)
			return
		}

		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")
		page := 1
		limit := 10
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > 100 {
				limit = 100
			} else {
				limit = l
			}
		}
		offset := (page - 1) * limit

		ctx := r.Context()
		out, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			IndexName:              aws.String("GSI1"),
			KeyConditionExpression: aws.String("GSI1PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S(store.UserPK(idUser)),
			},
			ScanIndexForward: aws.Bool(false),
		})
		if err != nil {
			http.Error(w, "Erro ao buscar doacoes: "+err.Error(), http.StatusInternalServerError)
			return
		}

		total := len(out.Items)
		end := offset + limit
		if offset > total {
			offset = total
		}
		if end > total {
			end = total
		}
		selected := out.Items[offset:end]

		keys := make([]map[string]types.AttributeValue, 0, len(selected)*2)
		for _, item := range selected {
			pk := item["PK"].(*types.AttributeValueMemberS).Value
			keys = append(keys, map[string]types.AttributeValue{"PK": dynamo.S(pk), "SK": dynamo.S("DETAILS")})
			keys = append(keys, map[string]types.AttributeValue{"PK": dynamo.S(pk), "SK": dynamo.S("PAYMENT")})
		}
		var batch map[string]map[string]types.AttributeValue
		if len(keys) > 0 {
			batch, _ = storeDDB.BatchGet(ctx, keys)
		}

		var donations []map[string]interface{}
		for _, item := range selected {
			var d map[string]interface{}
			attributevalue.UnmarshalMap(item, &d)

			pk := item["PK"].(*types.AttributeValueMemberS).Value
			detail := batch[pk+"|DETAILS"]
			payment := batch[pk+"|PAYMENT"]

			if detail != nil {
				var dd map[string]interface{}
				attributevalue.UnmarshalMap(detail, &dd)
				d["texto"] = dd["texto"]
				d["img"] = dd["img_caminho"]
				d["area"] = dd["area"]
			}
			if payment != nil {
				var pp map[string]interface{}
				attributevalue.UnmarshalMap(payment, &pp)
				d["pagamento"] = pp
			}

			donations = append(donations, d)
		}

		hasNext := end < total

		response := map[string]interface{}{
			"items":         donations,
			"page":          page,
			"limit":         limit,
			"total":         total,
			"has_next_page": hasNext,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func DonationDellHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		donationID := vars["id"]
		if donationID == "" {
			http.Error(w, "ID da doacao e obrigatorio na URL", http.StatusBadRequest)
			return
		}

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

		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			http.Error(w, "Token sem id_user", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		item, err := storeDDB.GetItem(ctx, store.DonationPK(donationID), "PROFILE")
		if err != nil || len(item) == 0 {
			http.Error(w, "Doacao nao encontrada", http.StatusNotFound)
			return
		}
		if v, ok := item["id_user"].(*types.AttributeValueMemberS); ok {
			if v.Value != userID {
				http.Error(w, "Usuario nao autorizado a deletar esta doacao", http.StatusForbidden)
				return
			}
		}

		err = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
			"PK": dynamo.S(store.DonationPK(donationID)),
			"SK": dynamo.S("PROFILE"),
		}, "SET dell = :d, date_update = :u", nil, map[string]types.AttributeValue{
			":d": dynamo.B(true),
			":u": dynamo.S(time.Now().Format(time.RFC3339)),
		})
		if err != nil {
			http.Error(w, "Erro ao deletar doacao: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Doacao deletada com sucesso",
		})
	}
}
