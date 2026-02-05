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

type ContactVisualizationRequest struct {
	Page      string `json:"page"`
	Timestamp string `json:"timestamp"`
	Referrer  string `json:"referrer"`
	Device    string `json:"device"`
	Language  string `json:"language"`
	IP        string `json:"ip"`
	User      string `json:"user"`
}

func ContactHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().UTC().Format(time.RFC3339)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message":  "online",
			"datetime": now,
		})
	}
}

func ContactVisualizationHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ContactVisualizationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao decodificar JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		page := strings.TrimSpace(req.Page)
		if page == "" {
			http.Error(w, "Campo page e obrigatorio", http.StatusBadRequest)
			return
		}

		ts := strings.TrimSpace(req.Timestamp)
		if ts == "" {
			ts = time.Now().UTC().Format(time.RFC3339)
		} else {
			if _, err := time.Parse(time.RFC3339, ts); err != nil {
				http.Error(w, "Campo timestamp invalido, use RFC3339", http.StatusBadRequest)
				return
			}
		}

		ip := strings.TrimSpace(req.IP)
		if ip == "" {
			ip = r.RemoteAddr
		}

		visID := uuid.NewString()
		item := map[string]types.AttributeValue{
			"PK":          dynamo.S(store.VisualizationPK(page)),
			"SK":          dynamo.S("VIS#" + ts + "#" + visID),
			"id":          dynamo.S(visID),
			"page":        dynamo.S(page),
			"timestamp":   dynamo.S(ts),
			"referrer":    dynamo.S(req.Referrer),
			"device":      dynamo.S(req.Device),
			"language":    dynamo.S(req.Language),
			"ip":          dynamo.S(ip),
			"user":        dynamo.S(req.User),
			"data_create": dynamo.S(time.Now().UTC().Format(time.RFC3339)),
		}

		if err := storeDDB.PutItem(r.Context(), item); err != nil {
			http.Error(w, "Erro ao salvar visualizacao: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Visualizacao registrada com sucesso",
			"id":      visID,
		})
	}
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
