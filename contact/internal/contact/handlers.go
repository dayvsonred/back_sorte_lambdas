package contact

import (
	"BACK_SORTE_GO/internal/store"
	"BACK_SORTE_GO/internal/store/dynamo"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

type ContactRequest struct {
	Nome     string `json:"nome"`
	Email    string `json:"email"`
	Mensagem string `json:"mensagem"`
	IP       string `json:"ip"`
	Location string `json:"location"`
	Token    string `json:"token"`
}

func ContactMensagemHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ContactRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao decodificar JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(req.Nome) == "" || strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Mensagem) == "" {
			http.Error(w, "Campos nome, email e mensagem sao obrigatorios", http.StatusBadRequest)
			return
		}

		if len(req.Mensagem) > 200 {
			http.Error(w, "A mensagem deve ter no maximo 200 caracteres", http.StatusBadRequest)
			return
		}

		id := uuid.NewString()
		dataCreate := time.Now().Format(time.RFC3339)

		item := map[string]types.AttributeValue{
			"PK":          dynamo.S(store.ContactPK(id)),
			"SK":          dynamo.S("DETAIL"),
			"id":          dynamo.S(id),
			"nome":        dynamo.S(req.Nome),
			"email":       dynamo.S(req.Email),
			"mensagem":    dynamo.S(req.Mensagem),
			"ip":          dynamo.S(req.IP),
			"location":    dynamo.S(req.Location),
			"token":       dynamo.S(req.Token),
			"view":        dynamo.B(false),
			"data_create": dynamo.S(dataCreate),
		}

		if err := storeDDB.PutItem(r.Context(), item); err != nil {
			http.Error(w, "Erro ao salvar mensagem: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Mensagem enviada com sucesso",
			"id":      id,
		})
	}
}
