package users

import (
	"BACK_SORTE_GO/internal/store"
	"BACK_SORTE_GO/internal/store/dynamo"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const emailConfirmMaxAttempts = 5

func UserConfirmEmailHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := strings.TrimSpace(r.URL.Query().Get("email"))
		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if email == "" || token == "" {
			http.Error(w, "Email e token sao obrigatorios", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		out, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			KeyConditionExpression: aws.String("PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S("EMAIL#VERIFY#" + token),
			},
			Limit: aws.Int32(1),
		})
		if err != nil {
			http.Error(w, "Erro ao buscar token de validacao", http.StatusInternalServerError)
			return
		}
		if len(out.Items) == 0 {
			http.Error(w, "Token invalido", http.StatusUnauthorized)
			return
		}

		item := out.Items[0]
		pkAttr, ok := item["PK"].(*types.AttributeValueMemberS)
		if !ok || strings.TrimSpace(pkAttr.Value) == "" {
			http.Error(w, "Token invalido", http.StatusUnauthorized)
			return
		}
		skAttr, ok := item["SK"].(*types.AttributeValueMemberS)
		if !ok || strings.TrimSpace(skAttr.Value) == "" {
			http.Error(w, "Token invalido", http.StatusUnauthorized)
			return
		}

		attempts := int64(0)
		if a, ok := item["attempts"].(*types.AttributeValueMemberN); ok {
			if parsed, parseErr := strconv.ParseInt(a.Value, 10, 64); parseErr == nil {
				attempts = parsed
			}
		}
		if b, ok := item["blocked"].(*types.AttributeValueMemberBOOL); ok && b.Value {
			http.Error(w, "Token bloqueado", http.StatusUnauthorized)
			return
		}
		if attempts >= emailConfirmMaxAttempts {
			_ = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
				"PK": dynamo.S(pkAttr.Value),
				"SK": dynamo.S(skAttr.Value),
			}, "SET blocked = :b, date_update = :d", nil, map[string]types.AttributeValue{
				":b": dynamo.B(true),
				":d": dynamo.S(time.Now().UTC().Format(time.RFC3339)),
			})
			http.Error(w, "Token bloqueado", http.StatusUnauthorized)
			return
		}
		if u, ok := item["used"].(*types.AttributeValueMemberBOOL); ok && u.Value {
			http.Error(w, "Token ja utilizado", http.StatusBadRequest)
			return
		}
		if exp, ok := item["expires_at"].(*types.AttributeValueMemberS); ok && strings.TrimSpace(exp.Value) != "" {
			expTime, parseErr := time.Parse(time.RFC3339, exp.Value)
			if parseErr != nil || time.Now().UTC().After(expTime) {
				http.Error(w, "Token expirado", http.StatusUnauthorized)
				return
			}
		}

		tokenEmail := ""
		if v, ok := item["email"].(*types.AttributeValueMemberS); ok {
			tokenEmail = strings.TrimSpace(v.Value)
		}
		if tokenEmail == "" || !strings.EqualFold(strings.ToLower(email), strings.ToLower(tokenEmail)) {
			newAttempts := attempts + 1
			blocked := newAttempts >= emailConfirmMaxAttempts
			_ = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
				"PK": dynamo.S(pkAttr.Value),
				"SK": dynamo.S(skAttr.Value),
			}, "SET attempts = :a, blocked = :b, date_update = :d", nil, map[string]types.AttributeValue{
				":a": dynamo.N(strconv.FormatInt(newAttempts, 10)),
				":b": dynamo.B(blocked),
				":d": dynamo.S(time.Now().UTC().Format(time.RFC3339)),
			})
			http.Error(w, "Email nao corresponde ao token informado", http.StatusUnauthorized)
			return
		}

		userID := ""
		if v, ok := item["user_id"].(*types.AttributeValueMemberS); ok {
			userID = strings.TrimSpace(v.Value)
		}
		if userID == "" {
			if strings.HasPrefix(skAttr.Value, "USER#") {
				userID = strings.TrimPrefix(skAttr.Value, "USER#")
			}
		}
		if userID == "" {
			http.Error(w, "Token invalido", http.StatusUnauthorized)
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		err = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
			"PK": dynamo.S(store.UserPK(userID)),
			"SK": dynamo.S("PROFILE"),
		}, "SET email_valid = :v, date_update = :d", nil, map[string]types.AttributeValue{
			":v": dynamo.B(true),
			":d": dynamo.S(now),
		})
		if err != nil {
			http.Error(w, "Erro ao atualizar validacao de email do usuario", http.StatusInternalServerError)
			return
		}

		err = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
			"PK": dynamo.S(pkAttr.Value),
			"SK": dynamo.S(skAttr.Value),
		}, "SET used = :u, attempts = :a, blocked = :b, date_update = :d", nil, map[string]types.AttributeValue{
			":u": dynamo.B(true),
			":a": dynamo.N(strconv.FormatInt(attempts, 10)),
			":b": dynamo.B(true),
			":d": dynamo.S(now),
		})
		if err != nil {
			http.Error(w, "Erro ao atualizar token de validacao", http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"message": "Email validado com sucesso",
		})
	}
}
