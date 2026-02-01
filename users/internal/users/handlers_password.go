package users

import (
	"BACK_SORTE_GO/config"
	"BACK_SORTE_GO/internal/store"
	"BACK_SORTE_GO/internal/store/dynamo"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const passwordRecoverTokenLength = 150
const passwordRecoverTokenChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateRandomToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = passwordRecoverTokenChars[int(b[i])%len(passwordRecoverTokenChars)]
	}
	return string(b), nil
}

func UserPasswordRecoverStartHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email string `json:"email"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao processar o JSON", http.StatusBadRequest)
			return
		}
		if req.Email == "" {
			http.Error(w, "Email e obrigatorio", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		userOut, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			IndexName:              aws.String("GSI2"),
			KeyConditionExpression: aws.String("GSI2PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S("EMAIL#" + strings.ToLower(req.Email)),
			},
			Limit: aws.Int32(1),
		})
		if err != nil || len(userOut.Items) == 0 {
			http.Error(w, "Email inexistente", http.StatusNotFound)
			return
		}
		var u userItem
		if err := attributevalue.UnmarshalMap(userOut.Items[0], &u); err != nil {
			http.Error(w, "Erro ao buscar usuario", http.StatusInternalServerError)
			return
		}

		today := time.Now().Format("2006-01-02")
		recoverOut, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			KeyConditionExpression: aws.String("PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S(store.PasswordPK(req.Email)),
			},
		})
		if err != nil {
			http.Error(w, "Erro ao verificar recuperacao: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for _, item := range recoverOut.Items {
			if v, ok := item["date_valid"].(*types.AttributeValueMemberS); ok && v.Value == today {
				if b, ok := item["blocked"].(*types.AttributeValueMemberBOOL); ok && !b.Value {
					http.Error(w, "Email ja enviado na data atual, por favor consulte seu email ou entre em contato com administradores", http.StatusBadRequest)
					return
				}
			}
		}

		for _, item := range recoverOut.Items {
			pk := item["PK"].(*types.AttributeValueMemberS).Value
			sk := item["SK"].(*types.AttributeValueMemberS).Value
			_ = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
				"PK": dynamo.S(pk),
				"SK": dynamo.S(sk),
			}, "SET blocked = :b", nil, map[string]types.AttributeValue{":b": dynamo.B(true)})
		}

		token, err := generateRandomToken(passwordRecoverTokenLength)
		if err != nil {
			http.Error(w, "Erro ao gerar token", http.StatusInternalServerError)
			return
		}

		recoverID := uuid.NewString()
		now := time.Now().Format(time.RFC3339)
		recoverItem := map[string]types.AttributeValue{
			"PK":          dynamo.S(store.PasswordPK(req.Email)),
			"SK":          dynamo.S("TS#" + now + "#" + recoverID),
			"id":          dynamo.S(recoverID),
			"id_user":     dynamo.S(u.ID),
			"email":       dynamo.S(req.Email),
			"token":       dynamo.S(token),
			"validated":   dynamo.B(false),
			"to_send":     dynamo.B(false),
			"attempt":     dynamo.N("0"),
			"blocked":     dynamo.B(false),
			"date_valid":  dynamo.S(today),
			"data_create": dynamo.S(now),
		}

		if err := storeDDB.PutItem(ctx, recoverItem); err != nil {
			http.Error(w, "Erro ao criar recuperacao: "+err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"message": "link de atualizacao de senha enviado para seu email. por favor verifique seu email",
		})
	}
}

func UserPasswordRecoverConfirmHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Token    string `json:"token"`
			Password string `json:"senha"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao processar o JSON", http.StatusBadRequest)
			return
		}
		if req.Email == "" || req.Token == "" || req.Password == "" {
			http.Error(w, "Email, token e senha sao obrigatorios", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		out, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			KeyConditionExpression: aws.String("PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S(store.PasswordPK(req.Email)),
			},
			ScanIndexForward: aws.Bool(false),
			Limit:            aws.Int32(1),
		})
		if err != nil || len(out.Items) == 0 {
			http.Error(w, "Recuperacao nao encontrada ou bloqueada", http.StatusNotFound)
			return
		}

		item := out.Items[0]
		storedToken, _ := item["token"].(*types.AttributeValueMemberS)
		attemptAttr, _ := item["attempt"].(*types.AttributeValueMemberN)
		attempt := int64(0)
		if attemptAttr != nil {
			fmt.Sscan(attemptAttr.Value, &attempt)
		}

		pk := item["PK"].(*types.AttributeValueMemberS).Value
		sk := item["SK"].(*types.AttributeValueMemberS).Value

		if storedToken == nil || storedToken.Value != req.Token {
			newAttempt := attempt + 1
			blocked := newAttempt > 5
			_ = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
				"PK": dynamo.S(pk),
				"SK": dynamo.S(sk),
			}, "SET attempt = :a, blocked = :b", nil, map[string]types.AttributeValue{
				":a": dynamo.N(fmt.Sprintf("%d", newAttempt)),
				":b": dynamo.B(blocked),
			})
			http.Error(w, "Token invalido", http.StatusUnauthorized)
			return
		}

		newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Erro ao criptografar senha", http.StatusInternalServerError)
			return
		}

		userID := ""
		if v, ok := item["id_user"].(*types.AttributeValueMemberS); ok {
			userID = v.Value
		}
		if userID == "" {
			http.Error(w, "Erro ao buscar usuario", http.StatusInternalServerError)
			return
		}

		err = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
			"PK": dynamo.S(store.UserPK(userID)),
			"SK": dynamo.S("PROFILE"),
		}, "SET password = :p, date_update = :d", nil, map[string]types.AttributeValue{
			":p": dynamo.S(string(newHashedPassword)),
			":d": dynamo.S(time.Now().Format(time.RFC3339)),
		})
		if err != nil {
			http.Error(w, "Erro ao atualizar a senha", http.StatusInternalServerError)
			return
		}

		_ = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
			"PK": dynamo.S(pk),
			"SK": dynamo.S(sk),
		}, "SET validated = :v, blocked = :b", nil, map[string]types.AttributeValue{
			":v": dynamo.B(true),
			":b": dynamo.B(true),
		})

		jsonResponse(w, http.StatusOK, map[string]string{
			"message": "Senha alterada com sucesso",
		})
	}
}

func UserPasswordChangeHandler(storeDDB *dynamo.Store) http.HandlerFunc {
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

		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			http.Error(w, "ID do usuario invalido no token", http.StatusUnauthorized)
			return
		}

		var req struct {
			OldPassword string `json:"old_password"`
			NewPassword string `json:"new_password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao processar o JSON", http.StatusBadRequest)
			return
		}
		if req.OldPassword == "" || req.NewPassword == "" {
			http.Error(w, "As senhas antiga e nova sao obrigatorias", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		item, err := storeDDB.GetItem(ctx, store.UserPK(userID), "PROFILE")
		if err != nil || len(item) == 0 {
			http.Error(w, "Usuario nao encontrado", http.StatusNotFound)
			return
		}
		var u userItem
		if err := attributevalue.UnmarshalMap(item, &u); err != nil {
			http.Error(w, "Usuario nao encontrado", http.StatusNotFound)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.OldPassword)); err != nil {
			http.Error(w, "Senha antiga incorreta", http.StatusUnauthorized)
			return
		}

		newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Erro ao criptografar nova senha", http.StatusInternalServerError)
			return
		}

		err = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
			"PK": dynamo.S(store.UserPK(userID)),
			"SK": dynamo.S("PROFILE"),
		}, "SET password = :p, date_update = :d", nil, map[string]types.AttributeValue{
			":p": dynamo.S(string(newHashedPassword)),
			":d": dynamo.S(time.Now().Format(time.RFC3339)),
		})
		if err != nil {
			http.Error(w, "Erro ao atualizar a senha", http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"message": "Senha atualizada com sucesso",
		})
	}
}

func UserPasswordRecoverLinkHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("email")
		key := r.URL.Query().Get("key")

		if email == "" || key == "" {
			http.Error(w, "Email e key sao obrigatorios", http.StatusBadRequest)
			return
		}

		if key != config.GetPasswordResetKey() {
			http.Error(w, "Key invalida", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		out, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			KeyConditionExpression: aws.String("PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S(store.PasswordPK(email)),
			},
			ScanIndexForward: aws.Bool(false),
			Limit:            aws.Int32(1),
		})
		if err != nil || len(out.Items) == 0 {
			http.Error(w, "Nenhum registro ativo encontrado para este email", http.StatusNotFound)
			return
		}

		item := out.Items[0]
		if b, ok := item["blocked"].(*types.AttributeValueMemberBOOL); ok && b.Value {
			http.Error(w, "Nenhum registro ativo encontrado para este email", http.StatusNotFound)
			return
		}
		if v, ok := item["validated"].(*types.AttributeValueMemberBOOL); ok && v.Value {
			http.Error(w, "Nenhum registro ativo encontrado para este email", http.StatusNotFound)
			return
		}

		tokenAttr, ok := item["token"].(*types.AttributeValueMemberS)
		if !ok || tokenAttr.Value == "" {
			http.Error(w, "Nenhum registro ativo encontrado para este email", http.StatusNotFound)
			return
		}

		link := fmt.Sprintf("http://localhost/auth/password-reset?token=%s&email=%s", tokenAttr.Value, email)
		mensagem := fmt.Sprintf("Clique aqui para resetar sua senha: %s", link)

		jsonResponse(w, http.StatusOK, map[string]string{
			"mensagem": mensagem,
		})
	}
}
