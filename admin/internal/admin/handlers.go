package admin

import (
	"BACK_SORTE_GO/config"
	"BACK_SORTE_GO/internal/store"
	"BACK_SORTE_GO/internal/store/dynamo"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	ddb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gorilla/mux"
)

const maxBatchWriteSize = 25

func DeleteUserHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authorizeAdmin(r); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		userID := mux.Vars(r)["id"]
		if strings.TrimSpace(userID) == "" {
			http.Error(w, "ID do usuario e obrigatorio", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		profile, err := storeDDB.GetItem(ctx, store.UserPK(userID), "PROFILE")
		if err != nil {
			http.Error(w, "Erro ao buscar usuario: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if len(profile) == 0 {
			http.Error(w, "Usuario nao encontrado", http.StatusNotFound)
			return
		}

		donationIDs, err := listDonationIDsByUser(ctx, storeDDB, userID)
		if err != nil {
			http.Error(w, "Erro ao listar doacoes do usuario: "+err.Error(), http.StatusInternalServerError)
			return
		}

		totalDonationItemsDeleted := 0
		for _, donationID := range donationIDs {
			deleted, found, err := deleteDonationData(ctx, storeDDB, donationID)
			if err != nil {
				http.Error(w, "Erro ao deletar doacao: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if found {
				totalDonationItemsDeleted += deleted
			}
		}

		userKeys, err := listKeysByPK(ctx, storeDDB, store.UserPK(userID))
		if err != nil {
			http.Error(w, "Erro ao listar dados do usuario: "+err.Error(), http.StatusInternalServerError)
			return
		}

		userDeleted, err := batchDeleteKeys(ctx, storeDDB, userKeys)
		if err != nil {
			http.Error(w, "Erro ao deletar dados do usuario: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message":                        "Usuario deletado com sucesso",
			"user_id":                        userID,
			"deleted_user_items":             userDeleted,
			"deleted_user_donation_count":    len(donationIDs),
			"deleted_user_donation_db_items": totalDonationItemsDeleted,
		})
	}
}

func DeleteDonationHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authorizeAdmin(r); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		donationID := mux.Vars(r)["id"]
		if strings.TrimSpace(donationID) == "" {
			http.Error(w, "ID da doacao e obrigatorio", http.StatusBadRequest)
			return
		}

		deleted, found, err := deleteDonationData(r.Context(), storeDDB, donationID)
		if err != nil {
			http.Error(w, "Erro ao deletar doacao: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			http.Error(w, "Doacao nao encontrada", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message":               "Doacao deletada com sucesso",
			"donation_id":           donationID,
			"deleted_donation_rows": deleted,
		})
	}
}

func BlockUserHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authorizeAdmin(r); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		userID := mux.Vars(r)["id"]
		if strings.TrimSpace(userID) == "" {
			http.Error(w, "ID do usuario e obrigatorio", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		profile, err := storeDDB.GetItem(ctx, store.UserPK(userID), "PROFILE")
		if err != nil {
			http.Error(w, "Erro ao buscar usuario: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if len(profile) == 0 {
			http.Error(w, "Usuario nao encontrado", http.StatusNotFound)
			return
		}

		err = storeDDB.UpdateItem(ctx,
			map[string]types.AttributeValue{
				"PK": dynamo.S(store.UserPK(userID)),
				"SK": dynamo.S("PROFILE"),
			},
			"SET blocked = :b, date_update = :du",
			nil,
			map[string]types.AttributeValue{
				":b":  dynamo.B(true),
				":du": dynamo.S(time.Now().Format(time.RFC3339)),
			},
		)
		if err != nil {
			http.Error(w, "Erro ao bloquear usuario: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Usuario bloqueado com sucesso",
			"user_id": userID,
		})
	}
}

func authorizeAdmin(r *http.Request) error {
	expected := strings.TrimSpace(config.GetAdminAuthKey())
	if expected == "" {
		return fmt.Errorf("ADMIN_AUTH_KEY nao configurada")
	}

	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return fmt.Errorf("Authorization nao fornecido")
	}

	if auth != expected && auth != "Bearer "+expected {
		return fmt.Errorf("Authorization invalido")
	}
	return nil
}

func listDonationIDsByUser(ctx context.Context, storeDDB *dynamo.Store, userID string) ([]string, error) {
	ids := make([]string, 0)
	seen := map[string]struct{}{}
	var startKey map[string]types.AttributeValue

	for {
		out, err := storeDDB.Query(ctx, &ddb.QueryInput{
			IndexName:              aws.String("GSI1"),
			KeyConditionExpression: aws.String("GSI1PK = :pk AND begins_with(GSI1SK, :pfx)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk":  dynamo.S(store.UserPK(userID)),
				":pfx": dynamo.S("DONATION#"),
			},
			ProjectionExpression: aws.String("#id"),
			ExpressionAttributeNames: map[string]string{
				"#id": "id",
			},
			ExclusiveStartKey: startKey,
		})
		if err != nil {
			return nil, err
		}

		for _, item := range out.Items {
			idAttr, ok := item["id"].(*types.AttributeValueMemberS)
			if !ok || strings.TrimSpace(idAttr.Value) == "" {
				continue
			}
			if _, exists := seen[idAttr.Value]; exists {
				continue
			}
			seen[idAttr.Value] = struct{}{}
			ids = append(ids, idAttr.Value)
		}

		if len(out.LastEvaluatedKey) == 0 {
			break
		}
		startKey = out.LastEvaluatedKey
	}

	return ids, nil
}

func deleteDonationData(ctx context.Context, storeDDB *dynamo.Store, donationID string) (int, bool, error) {
	profile, err := storeDDB.GetItem(ctx, store.DonationPK(donationID), "PROFILE")
	if err != nil {
		return 0, false, err
	}
	if len(profile) == 0 {
		return 0, false, nil
	}

	donationKeys, err := listKeysByPK(ctx, storeDDB, store.DonationPK(donationID))
	if err != nil {
		return 0, false, err
	}

	deleted, err := batchDeleteKeys(ctx, storeDDB, donationKeys)
	if err != nil {
		return 0, false, err
	}

	if linkAttr, ok := profile["nome_link"].(*types.AttributeValueMemberS); ok && strings.TrimSpace(linkAttr.Value) != "" {
		_, err := storeDDB.Client.DeleteItem(ctx, &ddb.DeleteItemInput{
			TableName: &storeDDB.Table,
			Key: map[string]types.AttributeValue{
				"PK": dynamo.S(store.LinkPK(linkAttr.Value)),
				"SK": dynamo.S("DONATION#" + donationID),
			},
		})
		if err != nil {
			return 0, false, err
		}
		deleted++
	}

	return deleted, true, nil
}

func listKeysByPK(ctx context.Context, storeDDB *dynamo.Store, pk string) ([]map[string]types.AttributeValue, error) {
	keys := make([]map[string]types.AttributeValue, 0)
	var startKey map[string]types.AttributeValue

	for {
		out, err := storeDDB.Query(ctx, &ddb.QueryInput{
			KeyConditionExpression: aws.String("PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S(pk),
			},
			ProjectionExpression: aws.String("PK, SK"),
			ExclusiveStartKey:    startKey,
		})
		if err != nil {
			return nil, err
		}

		for _, item := range out.Items {
			keys = append(keys, map[string]types.AttributeValue{
				"PK": item["PK"],
				"SK": item["SK"],
			})
		}

		if len(out.LastEvaluatedKey) == 0 {
			break
		}
		startKey = out.LastEvaluatedKey
	}

	return keys, nil
}

func batchDeleteKeys(ctx context.Context, storeDDB *dynamo.Store, keys []map[string]types.AttributeValue) (int, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	totalDeleted := 0
	for i := 0; i < len(keys); i += maxBatchWriteSize {
		end := i + maxBatchWriteSize
		if end > len(keys) {
			end = len(keys)
		}

		writeRequests := make([]types.WriteRequest, 0, end-i)
		for _, key := range keys[i:end] {
			writeRequests = append(writeRequests, types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{Key: key},
			})
		}

		pending := map[string][]types.WriteRequest{
			storeDDB.Table: writeRequests,
		}

		for retries := 0; retries < 5 && len(pending[storeDDB.Table]) > 0; retries++ {
			out, err := storeDDB.Client.BatchWriteItem(ctx, &ddb.BatchWriteItemInput{RequestItems: pending})
			if err != nil {
				return totalDeleted, err
			}
			pending = out.UnprocessedItems
			if len(pending[storeDDB.Table]) > 0 {
				time.Sleep(100 * time.Millisecond)
			}
		}

		if len(pending[storeDDB.Table]) > 0 {
			return totalDeleted, fmt.Errorf("nao foi possivel processar todas as delecoes em lote")
		}

		totalDeleted += len(writeRequests)
	}

	return totalDeleted, nil
}
