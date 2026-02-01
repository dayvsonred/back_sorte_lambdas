package pix

import (
	"BACK_SORTE_GO/config"
	"BACK_SORTE_GO/internal/store"
	"BACK_SORTE_GO/internal/store/dynamo"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/efipay/sdk-go-apis-efi/src/efipay/pix"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type PixChargeRequest struct {
	Valor    string `json:"valor"`
	CPF      string `json:"cpf"`
	Nome     string `json:"nome"`
	Chave    string `json:"chave"`
	Mensagem string `json:"mensagem"`
	Anonimo  bool   `json:"anonimo"`
	IdDoacao string `json:"id"`
}

func parseTimeISO(v interface{}) time.Time {
	if v == nil {
		return time.Now()
	}
	t, err := time.Parse(time.RFC3339, v.(string))
	if err != nil {
		return time.Now()
	}
	return t
}

func CreatePixTokenHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Metodo nao permitido", http.StatusMethodNotAllowed)
			return
		}

		var req PixChargeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao decodificar JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		efi := pix.NewEfiPay(config.GetCredentials())

		body := map[string]interface{}{
			"calendario": map[string]interface{}{"expiracao": 3600},
			"devedor": map[string]interface{}{
				"cpf":  req.CPF,
				"nome": req.Nome,
			},
			"valor":              map[string]interface{}{"original": req.Valor},
			"chave":              req.Chave,
			"solicitacaoPagador": "pagamento de doacao",
		}

		resStr, err := efi.CreateImmediateCharge(body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Erro ao criar cobranca PIX: %v", err), http.StatusInternalServerError)
			return
		}

		var resMap map[string]interface{}
		if err := json.Unmarshal([]byte(resStr), &resMap); err != nil {
			http.Error(w, "Erro ao decodificar resposta do PIX: "+err.Error(), http.StatusInternalServerError)
			return
		}

		txid, ok := resMap["txid"].(string)
		if !ok || txid == "" {
			http.Error(w, "Resposta invalida da API (txid ausente)", http.StatusInternalServerError)
			return
		}

		idPixQRCode := uuid.NewString()
		now := time.Now().Format(time.RFC3339)
		pixSK := store.PrefixPix + now + "#" + idPixQRCode

		pixItem := map[string]types.AttributeValue{
			"PK":           dynamo.S(store.DonationPK(req.IdDoacao)),
			"SK":           dynamo.S(pixSK),
			"id":           dynamo.S(idPixQRCode),
			"id_doacao":    dynamo.S(req.IdDoacao),
			"valor":        dynamo.N(req.Valor),
			"cpf":          dynamo.S(req.CPF),
			"nome":         dynamo.S(req.Nome),
			"mensagem":     dynamo.S(req.Mensagem),
			"anonimo":      dynamo.B(req.Anonimo),
			"visivel":      dynamo.B(false),
			"data_criacao": dynamo.S(now),
			"status":       dynamo.S(fmt.Sprint(resMap["status"])),
			"txid":         dynamo.S(txid),
		}

		statusItem := map[string]types.AttributeValue{
			"PK":               dynamo.S(store.TxPK(txid)),
			"SK":               dynamo.S("STATUS"),
			"id_pix_qrcode":    dynamo.S(idPixQRCode),
			"id_doacao":        dynamo.S(req.IdDoacao),
			"pix_sk":           dynamo.S(pixSK),
			"status":           dynamo.S(fmt.Sprint(resMap["status"])),
			"buscar":           dynamo.B(true),
			"finalizado":       dynamo.B(false),
			"data_pago":        dynamo.S(""),
			"expiracao":        dynamo.N(fmt.Sprint(resMap["calendario"].(map[string]interface{})["expiracao"])),
			"tipo_pagamento":   dynamo.S("v1"),
			"loc_id":           dynamo.N(fmt.Sprint(resMap["loc"].(map[string]interface{})["id"])),
			"loc_tipo_cob":     dynamo.S(fmt.Sprint(resMap["loc"].(map[string]interface{})["tipoCob"])),
			"loc_criacao":      dynamo.S(parseTimeISO(resMap["loc"].(map[string]interface{})["criacao"]).Format(time.RFC3339)),
			"location":         dynamo.S(fmt.Sprint(resMap["loc"].(map[string]interface{})["location"])),
			"pix_copia_e_cola": dynamo.S(fmt.Sprint(resMap["loc"].(map[string]interface{})["location"])),
			"chave":            dynamo.S(req.Chave),
			"id_pix":           dynamo.S(txid),
			"valor":            dynamo.N(req.Valor),
			"data_criacao":     dynamo.S(parseTimeISO(resMap["calendario"].(map[string]interface{})["criacao"]).Format(time.RFC3339)),
		}

		ctx := r.Context()
		err = storeDDB.TransactWrite(ctx, []types.TransactWriteItem{
			{Put: &types.Put{TableName: &storeDDB.Table, Item: pixItem}},
			{Put: &types.Put{TableName: &storeDDB.Table, Item: statusItem}},
		})
		if err != nil {
			http.Error(w, "Erro ao salvar pix: "+err.Error(), http.StatusInternalServerError)
			return
		}

		go func(txid string) {
			_ = IniciarMonitoramentoStatusPagamento(storeDDB, txid)
		}(txid)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resStr))
	}
}

func PixChargeStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		txid := vars["txid"]
		if txid == "" {
			http.Error(w, "txid e obrigatorio", http.StatusBadRequest)
			return
		}

		credentials := config.GetCredentials()
		efi := pix.NewEfiPay(credentials)
		res, err := efi.DetailCharge(txid)
		if err != nil {
			http.Error(w, fmt.Sprintf("Erro ao consultar status do PIX: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(res))
	}
}

func consultarStatusPix(txid string) (string, error) {
	efi := pix.NewEfiPay(config.GetCredentials())
	res, err := efi.DetailCharge(txid)
	if err != nil {
		return "", err
	}

	var resMap map[string]interface{}
	if err := json.Unmarshal([]byte(res), &resMap); err != nil {
		return "", err
	}

	status, ok := resMap["status"].(string)
	if !ok {
		return "", fmt.Errorf("status nao encontrado na resposta")
	}

	return status, nil
}

func IniciarMonitoramentoStatusPagamento(storeDDB *dynamo.Store, txid string) error {
	checkInterval := []time.Duration{30 * time.Second, 1 * time.Minute}
	attempts := []int{10, 21}

	for phase := 0; phase < 2; phase++ {
		for i := 0; i < attempts[phase]; i++ {
			status, err := consultarStatusPix(txid)
			if err != nil {
				return err
			}

			// MOCK: atualiza sempre
			_ = atualizarStatusPagamento(storeDDB, txid, status)
			return nil

			time.Sleep(checkInterval[phase])
		}
	}

	_ = marcarPagamentoVencido(storeDDB, txid)
	return nil
}

func MonitorarStatusPagamentoHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		txid := vars["txid"]
		if txid == "" {
			http.Error(w, "txid e obrigatorio", http.StatusBadRequest)
			return
		}

		go func() {
			_ = IniciarMonitoramentoStatusPagamento(storeDDB, txid)
		}()

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Monitoramento iniciado"))
	}
}

func atualizarStatusPagamento(storeDDB *dynamo.Store, txid, status string) error {
	now := time.Now().Format(time.RFC3339)

	err := storeDDB.UpdateItem(context.Background(), map[string]types.AttributeValue{
		"PK": dynamo.S(store.TxPK(txid)),
		"SK": dynamo.S("STATUS"),
	}, "SET #s = :s, buscar = :b, finalizado = :f, data_pago = :d", map[string]string{
		"#s": "status",
	}, map[string]types.AttributeValue{
		":s": dynamo.S("CONCLUIDA"),
		":b": dynamo.B(false),
		":f": dynamo.B(true),
		":d": dynamo.S(now),
	})
	if err != nil {
		return err
	}

	item, err := storeDDB.GetItem(context.Background(), store.TxPK(txid), "STATUS")
	if err != nil || len(item) == 0 {
		return err
	}

	idDoacao := ""
	pixSK := ""
	valor := "0"
	if v, ok := item["id_doacao"].(*types.AttributeValueMemberS); ok {
		idDoacao = v.Value
	}
	if v, ok := item["pix_sk"].(*types.AttributeValueMemberS); ok {
		pixSK = v.Value
	}
	if v, ok := item["valor"].(*types.AttributeValueMemberN); ok {
		valor = v.Value
	}

	if idDoacao == "" || pixSK == "" {
		return nil
	}

	_ = storeDDB.UpdateItem(context.Background(), map[string]types.AttributeValue{
		"PK": dynamo.S(store.DonationPK(idDoacao)),
		"SK": dynamo.S(pixSK),
	}, "SET visivel = :v, #s = :s", map[string]string{"#s": "status"}, map[string]types.AttributeValue{
		":v": dynamo.B(true),
		":s": dynamo.S("CONCLUIDA"),
	})

	valorFloat, _ := strconv.ParseFloat(valor, 64)
	valorLiquido := valorFloat * 0.90

	_ = storeDDB.UpdateItem(context.Background(), map[string]types.AttributeValue{
		"PK": dynamo.S(store.DonationPK(idDoacao)),
		"SK": dynamo.S("PAYMENT"),
	}, "SET valor_disponivel = if_not_exists(valor_disponivel, :z) + :v, data_update = :d", nil, map[string]types.AttributeValue{
		":z": dynamo.N("0"),
		":v": dynamo.N(fmt.Sprintf("%.2f", valorLiquido)),
		":d": dynamo.S(now),
	})

	return nil
}

func marcarPagamentoVencido(storeDDB *dynamo.Store, txid string) error {
	return storeDDB.UpdateItem(context.Background(), map[string]types.AttributeValue{
		"PK": dynamo.S(store.TxPK(txid)),
		"SK": dynamo.S("STATUS"),
	}, "SET #s = :s, buscar = :b", map[string]string{"#s": "status"}, map[string]types.AttributeValue{
		":s": dynamo.S("VENCIDO"),
		":b": dynamo.B(false),
	})
}

func MonitorarStatusAllPagamentosHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authKey := r.Header.Get("KEY")
		if authKey != "MINHAKEY_123" {
			http.Error(w, "Chave de acesso invalida", http.StatusUnauthorized)
			return
		}

		out, err := storeDDB.Scan(r.Context(), &dynamodb.ScanInput{
			FilterExpression: aws.String("begins_with(PK, :tx) AND #s = :st AND buscar = :b AND finalizado = :f"),
			ExpressionAttributeNames: map[string]string{
				"#s": "status",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":tx": dynamo.S(store.PrefixTx),
				":st": dynamo.S("ATIVA"),
				":b":  dynamo.B(true),
				":f":  dynamo.B(false),
			},
		})
		if err != nil {
			http.Error(w, "Erro ao buscar cobrancas ativas: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var txids []string
		for _, item := range out.Items {
			if v, ok := item["id_pix"].(*types.AttributeValueMemberS); ok {
				txids = append(txids, v.Value)
			}
		}

		for _, txid := range txids {
			go func(id string) {
				_ = IniciarMonitoramentoStatusPagamento(storeDDB, id)
			}(txid)
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":         "Monitoramento iniciado",
			"total_monitorar": len(txids),
		})
	}
}
