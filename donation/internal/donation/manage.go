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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func DonationClosedHandler(storeDDB *dynamo.Store) http.HandlerFunc {
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

		idUserToken, ok := claims["sub"].(string)
		if !ok || idUserToken == "" {
			http.Error(w, "ID do usuario invalido no token", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		donationID := vars["id"]
		if donationID == "" {
			http.Error(w, "ID da doacao e obrigatorio", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		item, err := storeDDB.GetItem(ctx, store.DonationPK(donationID), "PROFILE")
		if err != nil || len(item) == 0 {
			http.Error(w, "Doacao nao encontrada", http.StatusNotFound)
			return
		}
		if v, ok := item["id_user"].(*types.AttributeValueMemberS); ok {
			if v.Value != idUserToken {
				http.Error(w, "Voce nao tem permissao para encerrar esta doacao", http.StatusForbidden)
				return
			}
		}

		err = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
			"PK": dynamo.S(store.DonationPK(donationID)),
			"SK": dynamo.S("PROFILE"),
		}, "SET active = :a, closed = :c, date_update = :d", nil, map[string]types.AttributeValue{
			":a": dynamo.B(false),
			":c": dynamo.B(true),
			":d": dynamo.S(time.Now().Format(time.RFC3339)),
		})
		if err != nil {
			http.Error(w, "Erro ao encerrar doacao: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Doacao encerrada com sucesso",
		})
	}
}

func DonationRescueHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token nao fornecido", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey1, nil
		})
		if err != nil {
			http.Error(w, "Token invalido", http.StatusUnauthorized)
			return
		}

		idUser, ok := claims["sub"].(string)
		if !ok || idUser == "" {
			http.Error(w, "ID do usuario invalido", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		idDoacao := vars["id"]
		if idDoacao == "" {
			http.Error(w, "ID da doacao e obrigatorio", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		item, err := storeDDB.GetItem(ctx, store.DonationPK(idDoacao), "PROFILE")
		if err != nil || len(item) == 0 {
			http.Error(w, "Doacao nao encontrada", http.StatusNotFound)
			return
		}
		if v, ok := item["id_user"].(*types.AttributeValueMemberS); ok {
			if v.Value != idUser {
				http.Error(w, "Voce nao tem permissao para resgatar essa doacao", http.StatusForbidden)
				return
			}
		}

		out, err := storeDDB.Query(ctx, &dynamodb.QueryInput{
			KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S(store.DonationPK(idDoacao)),
				":sk": dynamo.S(store.PrefixPix),
			},
		})
		if err != nil {
			http.Error(w, "Erro ao calcular total recebido: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var totalValor float64
		for _, item := range out.Items {
			if st, ok := item["status"].(*types.AttributeValueMemberS); ok && st.Value == "CONCLUIDA" {
				if v, ok := item["valor"].(*types.AttributeValueMemberN); ok {
					val, _ := strconv.ParseFloat(v.Value, 64)
					totalValor += val
				}
			}
		}

		if totalValor <= 0 {
			http.Error(w, "Nenhum valor disponivel para resgate", http.StatusBadRequest)
			return
		}

		valorDisponivel := totalValor * 0.90
		dataSolicitado := time.Now().Format(time.RFC3339)

		err = storeDDB.UpdateItem(ctx, map[string]types.AttributeValue{
			"PK": dynamo.S(store.DonationPK(idDoacao)),
			"SK": dynamo.S("PAYMENT"),
		}, "SET valor_disponivel = :v, data_solicitado = :d, status = :s, solicitado = :b, data_update = :u", nil, map[string]types.AttributeValue{
			":v": dynamo.N(fmt.Sprintf("%.2f", valorDisponivel)),
			":d": dynamo.S(dataSolicitado),
			":s": dynamo.S("PROCESS"),
			":b": dynamo.B(true),
			":u": dynamo.S(time.Now().Format(time.RFC3339)),
		})
		if err != nil {
			http.Error(w, "Erro ao atualizar pagamento: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":          "Resgate processado com sucesso",
			"valor_disponivel": valorDisponivel,
			"resgate_total":    totalValor,
		})
	}
}

type DonationVisualizationRequest struct {
	IDDoacao       string `json:"id_doacao"`
	IDUser         string `json:"id_user"`
	Visuaization   bool   `json:"visuaization"`
	Idioma         string `json:"idioma"`
	Tema           string `json:"tema"`
	Form           string `json:"form"`
	Google         string `json:"google"`
	GoogleMaps     string `json:"google_maps"`
	GoogleAds      string `json:"google_ads"`
	MetaPixel      string `json:"meta_pixel"`
	CookiesStripe  string `json:"Cookies_Stripe"`
	CookiesPayPal  string `json:"Cookies_PayPal"`
	VisitorInfo    string `json:"visitor_info1_live"`
	DonationLike   bool   `json:"donation_like"`
	Love           bool   `json:"love"`
	Shared         bool   `json:"shared"`
	AcesseDonation bool   `json:"acesse_donation"`
	CreatePix      bool   `json:"create_pix"`
	CreateCartao   bool   `json:"create_cartao"`
	CreatePayPal   bool   `json:"create_paypal"`
	CreateGoogle   bool   `json:"create_google"`
	CreatePag1     bool   `json:"create_pag1"`
	CreatePag2     bool   `json:"create_pag2"`
	CreatePag3     bool   `json:"create_pag3"`
}

func DonationVisualization(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req DonationVisualizationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		aggKey := map[string]types.AttributeValue{
			"PK": dynamo.S(store.DonationPK(req.IDDoacao)),
			"SK": dynamo.S("VISUALIZATION"),
		}

		updateParts := []string{}
		values := map[string]types.AttributeValue{}
		if req.Visuaization {
			updateParts = append(updateParts, "visualization = if_not_exists(visualization, :z) + :one")
			values[":one"] = dynamo.N("1")
			values[":z"] = dynamo.N("0")
		}
		if req.DonationLike {
			updateParts = append(updateParts, "donation_like = if_not_exists(donation_like, :z) + :one")
			values[":one"] = dynamo.N("1")
			values[":z"] = dynamo.N("0")
		}
		if req.Love {
			updateParts = append(updateParts, "love = if_not_exists(love, :z) + :one")
			values[":one"] = dynamo.N("1")
			values[":z"] = dynamo.N("0")
		}
		if req.Shared {
			updateParts = append(updateParts, "shared = if_not_exists(shared, :z) + :one")
			values[":one"] = dynamo.N("1")
			values[":z"] = dynamo.N("0")
		}
		if req.AcesseDonation {
			updateParts = append(updateParts, "acesse_donation = if_not_exists(acesse_donation, :z) + :one")
			values[":one"] = dynamo.N("1")
			values[":z"] = dynamo.N("0")
		}
		if req.CreatePix {
			updateParts = append(updateParts, "create_pix = if_not_exists(create_pix, :z) + :one")
			values[":one"] = dynamo.N("1")
			values[":z"] = dynamo.N("0")
		}
		if req.CreateCartao {
			updateParts = append(updateParts, "create_cartao = if_not_exists(create_cartao, :z) + :one")
			values[":one"] = dynamo.N("1")
			values[":z"] = dynamo.N("0")
		}
		if req.CreatePayPal {
			updateParts = append(updateParts, "create_paypal = if_not_exists(create_paypal, :z) + :one")
			values[":one"] = dynamo.N("1")
			values[":z"] = dynamo.N("0")
		}
		if req.CreateGoogle {
			updateParts = append(updateParts, "create_google = if_not_exists(create_google, :z) + :one")
			values[":one"] = dynamo.N("1")
			values[":z"] = dynamo.N("0")
		}
		if req.CreatePag1 {
			updateParts = append(updateParts, "create_pag1 = if_not_exists(create_pag1, :z) + :one")
			values[":one"] = dynamo.N("1")
			values[":z"] = dynamo.N("0")
		}
		if req.CreatePag2 {
			updateParts = append(updateParts, "create_pag2 = if_not_exists(create_pag2, :z) + :one")
			values[":one"] = dynamo.N("1")
			values[":z"] = dynamo.N("0")
		}
		if req.CreatePag3 {
			updateParts = append(updateParts, "create_pag3 = if_not_exists(create_pag3, :z) + :one")
			values[":one"] = dynamo.N("1")
			values[":z"] = dynamo.N("0")
		}

		if len(updateParts) > 0 {
			updateExpr := "SET " + strings.Join(updateParts, ", ") + ", date_update = :u"
			values[":u"] = dynamo.S(time.Now().Format(time.RFC3339))
			_ = storeDDB.UpdateItem(ctx, aggKey, updateExpr, nil, values)
		}

		visID := uuid.NewString()
		visItem := map[string]types.AttributeValue{
			"PK":                 dynamo.S(store.DonationPK(req.IDDoacao)),
			"SK":                 dynamo.S("VIS#" + time.Now().Format(time.RFC3339) + "#" + visID),
			"id":                 dynamo.S(visID),
			"id_visualization":   dynamo.S("VISUALIZATION"),
			"ip":                 dynamo.S(r.RemoteAddr),
			"id_user":            dynamo.S(req.IDUser),
			"idioma":             dynamo.S(req.Idioma),
			"tema":               dynamo.S(req.Tema),
			"form":               dynamo.S(req.Form),
			"google":             dynamo.S(req.Google),
			"google_maps":        dynamo.S(req.GoogleMaps),
			"google_ads":         dynamo.S(req.GoogleAds),
			"meta_pixel":         dynamo.S(req.MetaPixel),
			"Cookies_Stripe":     dynamo.S(req.CookiesStripe),
			"Cookies_PayPal":     dynamo.S(req.CookiesPayPal),
			"visitor_info1_live": dynamo.S(req.VisitorInfo),
			"donation_like":      dynamo.B(req.DonationLike),
			"love":               dynamo.B(req.Love),
			"shared":             dynamo.B(req.Shared),
			"acesse_donation":    dynamo.B(req.AcesseDonation),
			"create_pix":         dynamo.B(req.CreatePix),
			"create_cartao":      dynamo.B(req.CreateCartao),
			"create_paypal":      dynamo.B(req.CreatePayPal),
			"create_google":      dynamo.B(req.CreateGoogle),
			"create_pag1":        dynamo.B(req.CreatePag1),
			"create_pag2":        dynamo.B(req.CreatePag2),
			"create_pag3":        dynamo.B(req.CreatePag3),
			"date_create":        dynamo.S(time.Now().Format(time.RFC3339)),
		}
		_ = storeDDB.PutItem(ctx, visItem)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Visualizacao registrada com sucesso",
		})
	}
}
