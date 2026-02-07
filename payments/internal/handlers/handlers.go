package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"BACK_SORTE_GO/internal/config"
	"BACK_SORTE_GO/internal/dynamo"
	"BACK_SORTE_GO/internal/models"
	"BACK_SORTE_GO/internal/stripeclient"
	"BACK_SORTE_GO/internal/utils"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v78"
)

type Handler struct {
	Store  *dynamo.Store
	Stripe *stripeclient.Client
	Cfg    config.Config
	Log    *utils.Logger
}

func NewHandler(store *dynamo.Store, stripeClient *stripeclient.Client, cfg config.Config, logger *utils.Logger) *Handler {
	return &Handler{
		Store:  store,
		Stripe: stripeClient,
		Cfg:    cfg,
		Log:    logger,
	}
}

type createDonationRequest struct {
	CampaignID string `json:"campaignId"`
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	Donor      struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"donor"`
}

type createIntentRequest struct {
	DonationID string `json:"donationId"`
}

type createCheckoutSessionRequest struct {
	CampaignID string `json:"campaignId"`
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	SuccessURL string `json:"successUrl"`
	CancelURL  string `json:"cancelUrl"`
	Donor      struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"donor"`
}

func (h *Handler) CreateDonation(w http.ResponseWriter, r *http.Request) {
	var req createDonationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "JSON invalido")
		return
	}

	campaignID := strings.TrimSpace(req.CampaignID)
	if campaignID == "" {
		utils.RespondError(w, http.StatusBadRequest, "campaignId e obrigatorio")
		return
	}

	amountCents, err := utils.ParseAmountToCents(req.Amount)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "amount invalido: "+err.Error())
		return
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "BRL"
	}
	if currency != "BRL" {
		utils.RespondError(w, http.StatusBadRequest, "currency deve ser BRL")
		return
	}

	donorName := strings.TrimSpace(req.Donor.Name)
	donorEmail := strings.TrimSpace(req.Donor.Email)
	if donorName == "" || donorEmail == "" {
		utils.RespondError(w, http.StatusBadRequest, "donor.name e donor.email sao obrigatorios")
		return
	}

	donationID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)

	item := map[string]types.AttributeValue{
		"PK":             dynamo.S("DONATION#" + donationID),
		"SK":             dynamo.S("DONATION#" + donationID),
		"donationId":     dynamo.S(donationID),
		"campaignId":     dynamo.S(campaignID),
		"amountExpected": dynamo.N(intToString(amountCents)),
		"currency":       dynamo.S(currency),
		"status":         dynamo.S(string(models.DonationStatusCreated)),
		"donorName":      dynamo.S(donorName),
		"donorEmail":     dynamo.S(donorEmail),
		"createdAt":      dynamo.S(now),
		"updatedAt":      dynamo.S(now),
	}

	if err := h.Store.PutItem(r.Context(), item); err != nil {
		h.Log.Error("erro_ao_salvar_donation", map[string]interface{}{"error": err.Error()})
		utils.RespondError(w, http.StatusInternalServerError, "erro ao salvar donation")
		return
	}

	h.Log.Info("donation_criada", map[string]interface{}{"donationId": donationID, "campaignId": campaignID})
	utils.RespondJSON(w, http.StatusCreated, map[string]string{
		"donationId": donationID,
	})
}

func (h *Handler) CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	var req createIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "JSON invalido")
		return
	}

	donationID := strings.TrimSpace(req.DonationID)
	if donationID == "" {
		utils.RespondError(w, http.StatusBadRequest, "donationId e obrigatorio")
		return
	}

	donationItem, err := h.Store.GetItem(r.Context(), "DONATION#"+donationID, "DONATION#"+donationID)
	if err != nil {
		h.Log.Error("erro_ao_buscar_donation", map[string]interface{}{"error": err.Error(), "donationId": donationID})
		utils.RespondError(w, http.StatusInternalServerError, "erro ao buscar donation")
		return
	}
	if len(donationItem) == 0 {
		utils.RespondError(w, http.StatusNotFound, "donation nao encontrada")
		return
	}

	campaignID := getStringAttr(donationItem, "campaignId")
	currency := strings.ToUpper(getStringAttr(donationItem, "currency"))
	status := getStringAttr(donationItem, "status")
	if status == string(models.DonationStatusPaid) {
		utils.RespondError(w, http.StatusConflict, "donation ja paga")
		return
	}
	amountExpectedStr := getNumberAttr(donationItem, "amountExpected")
	amountExpected, err := parseInt64(amountExpectedStr)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "amountExpected invalido")
		return
	}

	pi, err := h.Stripe.CreatePaymentIntent(r.Context(), amountExpected, strings.ToLower(currency), donationID, campaignID)
	if err != nil {
		h.Log.Error("erro_criar_payment_intent", map[string]interface{}{"error": err.Error(), "donationId": donationID})
		utils.RespondError(w, http.StatusBadGateway, "erro ao criar payment intent")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	createdAtStripe := time.Unix(pi.Created, 0).UTC().Format(time.RFC3339)
	paymentItem := map[string]types.AttributeValue{
		"PK":              dynamo.S("PAYMENT#" + pi.ID),
		"SK":              dynamo.S("DONATION#" + donationID),
		"paymentIntentId": dynamo.S(pi.ID),
		"donationId":      dynamo.S(donationID),
		"campaignId":      dynamo.S(campaignID),
		"amount":          dynamo.N(intToString(amountExpected)),
		"currency":        dynamo.S(currency),
		"status":          dynamo.S(string(models.PaymentStatusPending)),
		"createdAtStripe": dynamo.S(createdAtStripe),
		"createdAt":       dynamo.S(now),
		"updatedAt":       dynamo.S(now),
	}

	updateDonation := types.TransactWriteItem{
		Update: &types.Update{
			TableName: aws.String(h.Store.TableName()),
			Key: map[string]types.AttributeValue{
				"PK": dynamo.S("DONATION#" + donationID),
				"SK": dynamo.S("DONATION#" + donationID),
			},
			UpdateExpression: aws.String("SET #status = :status, #updatedAt = :updatedAt"),
			ExpressionAttributeNames: map[string]string{
				"#status":    "status",
				"#updatedAt": "updatedAt",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":status":    dynamo.S(string(models.DonationStatusPendingPayment)),
				":updatedAt": dynamo.S(now),
			},
		},
	}

	putPayment := types.TransactWriteItem{
		Put: &types.Put{
			TableName: aws.String(h.Store.TableName()),
			Item:      paymentItem,
		},
	}

	if err := h.Store.TransactWrite(r.Context(), []types.TransactWriteItem{putPayment, updateDonation}); err != nil {
		h.Log.Error("erro_ao_salvar_payment", map[string]interface{}{"error": err.Error(), "paymentIntentId": pi.ID})
		utils.RespondError(w, http.StatusInternalServerError, "erro ao salvar payment")
		return
	}

	h.Log.Info("payment_intent_criado", map[string]interface{}{"donationId": donationID, "paymentIntentId": pi.ID})
	utils.RespondJSON(w, http.StatusOK, map[string]string{
		"client_secret":   pi.ClientSecret,
		"paymentIntentId": pi.ID,
	})
}

func (h *Handler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	var req createCheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "JSON invalido")
		return
	}

	campaignID := strings.TrimSpace(req.CampaignID)
	if campaignID == "" {
		utils.RespondError(w, http.StatusBadRequest, "campaignId e obrigatorio")
		return
	}

	amountCents, err := utils.ParseAmountToCents(req.Amount)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "amount invalido: "+err.Error())
		return
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "BRL"
	}
	if currency != "BRL" {
		utils.RespondError(w, http.StatusBadRequest, "currency deve ser BRL")
		return
	}

	successURL := strings.TrimSpace(req.SuccessURL)
	cancelURL := strings.TrimSpace(req.CancelURL)
	if successURL == "" || cancelURL == "" {
		utils.RespondError(w, http.StatusBadRequest, "successUrl e cancelUrl sao obrigatorios")
		return
	}

	donorName := strings.TrimSpace(req.Donor.Name)
	donorEmail := strings.TrimSpace(req.Donor.Email)
	if donorName == "" || donorEmail == "" {
		utils.RespondError(w, http.StatusBadRequest, "donor.name e donor.email sao obrigatorios")
		return
	}

	donationID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)

	item := map[string]types.AttributeValue{
		"PK":             dynamo.S("DONATION#" + donationID),
		"SK":             dynamo.S("DONATION#" + donationID),
		"donationId":     dynamo.S(donationID),
		"campaignId":     dynamo.S(campaignID),
		"amountExpected": dynamo.N(intToString(amountCents)),
		"currency":       dynamo.S(currency),
		"status":         dynamo.S(string(models.DonationStatusCreated)),
		"donorName":      dynamo.S(donorName),
		"donorEmail":     dynamo.S(donorEmail),
		"createdAt":      dynamo.S(now),
		"updatedAt":      dynamo.S(now),
	}

	if err := h.Store.PutItem(r.Context(), item); err != nil {
		h.Log.Error("erro_ao_salvar_donation", map[string]interface{}{"error": err.Error()})
		utils.RespondError(w, http.StatusInternalServerError, "erro ao salvar donation")
		return
	}

	session, err := h.Stripe.CreateCheckoutSession(
		r.Context(),
		amountCents,
		strings.ToLower(currency),
		donationID,
		campaignID,
		donorName,
		donorEmail,
		successURL,
		cancelURL,
	)
	if err != nil {
		h.Log.Error("erro_criar_checkout_session", map[string]interface{}{"error": err.Error(), "donationId": donationID})
		utils.RespondError(w, http.StatusBadGateway, "erro ao criar checkout session")
		return
	}

	paymentIntentID := ""
	if session.PaymentIntent != nil {
		paymentIntentID = session.PaymentIntent.ID
	}
	if paymentIntentID == "" {
		h.Log.Error("checkout_sem_payment_intent", map[string]interface{}{"donationId": donationID, "sessionId": session.ID})
		utils.RespondError(w, http.StatusBadGateway, "checkout session sem payment intent")
		return
	}

	createdAtStripe := time.Unix(session.Created, 0).UTC().Format(time.RFC3339)
	paymentItem := map[string]types.AttributeValue{
		"PK":              dynamo.S("PAYMENT#" + paymentIntentID),
		"SK":              dynamo.S("DONATION#" + donationID),
		"paymentIntentId": dynamo.S(paymentIntentID),
		"donationId":      dynamo.S(donationID),
		"campaignId":      dynamo.S(campaignID),
		"amount":          dynamo.N(intToString(amountCents)),
		"currency":        dynamo.S(currency),
		"status":          dynamo.S(string(models.PaymentStatusPending)),
		"createdAtStripe": dynamo.S(createdAtStripe),
		"createdAt":       dynamo.S(now),
		"updatedAt":       dynamo.S(now),
	}

	updateDonation := types.TransactWriteItem{
		Update: &types.Update{
			TableName: aws.String(h.Store.TableName()),
			Key: map[string]types.AttributeValue{
				"PK": dynamo.S("DONATION#" + donationID),
				"SK": dynamo.S("DONATION#" + donationID),
			},
			UpdateExpression: aws.String("SET #status = :status, #updatedAt = :updatedAt"),
			ExpressionAttributeNames: map[string]string{
				"#status":    "status",
				"#updatedAt": "updatedAt",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":status":    dynamo.S(string(models.DonationStatusPendingPayment)),
				":updatedAt": dynamo.S(now),
			},
		},
	}

	putPayment := types.TransactWriteItem{
		Put: &types.Put{
			TableName: aws.String(h.Store.TableName()),
			Item:      paymentItem,
		},
	}

	if err := h.Store.TransactWrite(r.Context(), []types.TransactWriteItem{putPayment, updateDonation}); err != nil {
		h.Log.Error("erro_ao_salvar_payment", map[string]interface{}{"error": err.Error(), "paymentIntentId": paymentIntentID})
		utils.RespondError(w, http.StatusInternalServerError, "erro ao salvar payment")
		return
	}

	h.Log.Info("checkout_session_criada", map[string]interface{}{"donationId": donationID, "sessionId": session.ID})
	utils.RespondJSON(w, http.StatusOK, map[string]string{
		"url":             session.URL,
		"donationId":      donationID,
		"paymentIntentId": paymentIntentID,
	})
}

func (h *Handler) HandleEventBridge(ctx context.Context, ebEvent events.EventBridgeEvent) (map[string]string, error) {
	h.Log.Info("stripe_eventbridge_recebido", map[string]interface{}{
		"id":         ebEvent.ID,
		"source":     ebEvent.Source,
		"detailType": ebEvent.DetailType,
		"time":       ebEvent.Time.Format(time.RFC3339),
	})

	var stripeEvent stripe.Event
	if err := json.Unmarshal(ebEvent.Detail, &stripeEvent); err != nil {
		h.Log.Error("eventbridge_detail_invalido", map[string]interface{}{"error": err.Error()})
		return map[string]string{"status": "invalid"}, nil
	}

	h.Log.Info("stripe_evento_recebido", map[string]interface{}{
		"eventId":   stripeEvent.ID,
		"eventType": string(stripeEvent.Type),
		"created":   stripeEvent.Created,
	})

	switch stripeEvent.Type {
	case "payment_intent.succeeded":
		return h.handleStripeEvent(ctx, stripeEvent, models.PaymentStatusSucceeded)
	case "payment_intent.payment_failed":
		return h.handleStripeEvent(ctx, stripeEvent, models.PaymentStatusFailed)
	default:
		return map[string]string{"status": "ignored"}, nil
	}
}

func (h *Handler) handleStripeEvent(ctx context.Context, event stripe.Event, status models.PaymentStatus) (map[string]string, error) {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		h.Log.Error("evento_invalido", map[string]interface{}{"error": err.Error(), "eventId": event.ID})
		return map[string]string{"status": "invalid"}, nil
	}

	donationID := strings.TrimSpace(pi.Metadata["donationId"])
	h.Log.Info("stripe_payment_intent_parseado", map[string]interface{}{
		"paymentIntentId": pi.ID,
		"donationId":      donationID,
		"amount":          pi.Amount,
		"currency":        string(pi.Currency),
	})
	if donationID == "" {
		h.Log.Info("donation_id_ausente_evento", map[string]interface{}{"eventId": event.ID, "paymentIntentId": pi.ID})
		return h.handleStripeEventWithoutMetadata(ctx, event, pi, status)
	}

	campaignID := strings.TrimSpace(pi.Metadata["campaignId"])
	now := time.Now().UTC().Format(time.RFC3339)
	eventCreated := time.Unix(event.Created, 0).UTC().Format(time.RFC3339)
	chargeID := ""
	if pi.LatestCharge != nil {
		chargeID = pi.LatestCharge.ID
	}

	paymentStatus := status
	donationStatus := models.DonationStatusFailed
	if status == models.PaymentStatusSucceeded {
		donationStatus = models.DonationStatusPaid
	}
	h.Log.Info("stripe_evento_processando", map[string]interface{}{
		"eventId":         event.ID,
		"paymentIntentId": pi.ID,
		"donationId":      donationID,
		"campaignId":      campaignID,
		"status":          string(paymentStatus),
	})

	eventItem := map[string]types.AttributeValue{
		"PK":              dynamo.S("EVENT#" + event.ID),
		"SK":              dynamo.S("EVENT#" + event.ID),
		"eventId":         dynamo.S(event.ID),
		"eventType":       dynamo.S(string(event.Type)),
		"paymentIntentId": dynamo.S(pi.ID),
		"donationId":      dynamo.S(donationID),
		"createdAt":       dynamo.S(now),
	}

	paymentUpdateExpr := "SET #status = :status, #updatedAt = :updatedAt, #rawEventLastId = :eventId"
	paymentNames := map[string]string{
		"#status":         "status",
		"#updatedAt":      "updatedAt",
		"#rawEventLastId": "rawEventLastId",
	}
	paymentValues := map[string]types.AttributeValue{
		":status":    dynamo.S(string(paymentStatus)),
		":updatedAt": dynamo.S(now),
		":eventId":   dynamo.S(event.ID),
	}

	if status == models.PaymentStatusSucceeded {
		paymentUpdateExpr += ", #succeededAtStripe = :succeededAtStripe"
		paymentNames["#succeededAtStripe"] = "succeededAtStripe"
		paymentValues[":succeededAtStripe"] = dynamo.S(eventCreated)
	}
	if chargeID != "" {
		paymentUpdateExpr += ", #chargeId = :chargeId"
		paymentNames["#chargeId"] = "chargeId"
		paymentValues[":chargeId"] = dynamo.S(chargeID)
	}

	paymentUpdate := types.TransactWriteItem{
		Update: &types.Update{
			TableName: aws.String(h.Store.TableName()),
			Key: map[string]types.AttributeValue{
				"PK": dynamo.S("PAYMENT#" + pi.ID),
				"SK": dynamo.S("DONATION#" + donationID),
			},
			UpdateExpression:          aws.String(paymentUpdateExpr),
			ExpressionAttributeNames:  paymentNames,
			ExpressionAttributeValues: paymentValues,
			ConditionExpression:       aws.String("attribute_exists(PK)"),
		},
	}

	donationUpdate := types.TransactWriteItem{
		Update: &types.Update{
			TableName: aws.String(h.Store.TableName()),
			Key: map[string]types.AttributeValue{
				"PK": dynamo.S("DONATION#" + donationID),
				"SK": dynamo.S("DONATION#" + donationID),
			},
			UpdateExpression: aws.String("SET #status = :status, #updatedAt = :updatedAt"),
			ExpressionAttributeNames: map[string]string{
				"#status":    "status",
				"#updatedAt": "updatedAt",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":status":    dynamo.S(string(donationStatus)),
				":updatedAt": dynamo.S(now),
			},
			ConditionExpression: aws.String("attribute_exists(PK)"),
		},
	}

	eventPut := types.TransactWriteItem{
		Put: &types.Put{
			TableName:           aws.String(h.Store.TableName()),
			Item:                eventItem,
			ConditionExpression: aws.String("attribute_not_exists(PK)"),
		},
	}

	if err := h.Store.TransactWrite(ctx, []types.TransactWriteItem{eventPut, paymentUpdate, donationUpdate}); err != nil {
		if isConditionalCheckFailed(err) {
			h.Log.Info("evento_ja_processado", map[string]interface{}{"eventId": event.ID})
			return map[string]string{"status": "ok"}, nil
		}
		h.Log.Error("erro_processar_evento", map[string]interface{}{"error": err.Error(), "eventId": event.ID, "paymentIntentId": pi.ID})
		return map[string]string{"status": "error"}, err
	}

	h.Log.Info("evento_processado", map[string]interface{}{
		"eventId":         event.ID,
		"paymentIntentId": pi.ID,
		"donationId":      donationID,
		"campaignId":      campaignID,
		"status":          string(paymentStatus),
		"amount":          pi.Amount,
		"currency":        string(pi.Currency),
	})
	return map[string]string{"status": "ok"}, nil
}

func (h *Handler) handleStripeEventWithoutMetadata(ctx context.Context, event stripe.Event, pi stripe.PaymentIntent, status models.PaymentStatus) (map[string]string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	eventCreated := time.Unix(event.Created, 0).UTC().Format(time.RFC3339)
	chargeID := ""
	if pi.LatestCharge != nil {
		chargeID = pi.LatestCharge.ID
	}

	paymentStatus := status
	sk := "DONATION#UNKNOWN"

	eventItem := map[string]types.AttributeValue{
		"PK":              dynamo.S("EVENT#" + event.ID),
		"SK":              dynamo.S("EVENT#" + event.ID),
		"eventId":         dynamo.S(event.ID),
		"eventType":       dynamo.S(string(event.Type)),
		"paymentIntentId": dynamo.S(pi.ID),
		"createdAt":       dynamo.S(now),
	}

	updateExpr := "SET #status = :status, #updatedAt = :updatedAt, #rawEventLastId = :eventId, #amount = :amount, #currency = :currency, #createdAtStripe = :createdAtStripe"
	names := map[string]string{
		"#status":          "status",
		"#updatedAt":       "updatedAt",
		"#rawEventLastId":  "rawEventLastId",
		"#amount":          "amount",
		"#currency":        "currency",
		"#createdAtStripe": "createdAtStripe",
	}
	values := map[string]types.AttributeValue{
		":status":          dynamo.S(string(paymentStatus)),
		":updatedAt":       dynamo.S(now),
		":eventId":         dynamo.S(event.ID),
		":amount":          dynamo.N(intToString(pi.Amount)),
		":currency":        dynamo.S(strings.ToUpper(string(pi.Currency))),
		":createdAtStripe": dynamo.S(eventCreated),
	}
	if status == models.PaymentStatusSucceeded {
		updateExpr += ", #succeededAtStripe = :succeededAtStripe"
		names["#succeededAtStripe"] = "succeededAtStripe"
		values[":succeededAtStripe"] = dynamo.S(eventCreated)
	}
	if chargeID != "" {
		updateExpr += ", #chargeId = :chargeId"
		names["#chargeId"] = "chargeId"
		values[":chargeId"] = dynamo.S(chargeID)
	}
	updateExpr += ", #createdAt = if_not_exists(#createdAt, :createdAt)"
	names["#createdAt"] = "createdAt"
	values[":createdAt"] = dynamo.S(now)

	paymentUpdate := types.TransactWriteItem{
		Update: &types.Update{
			TableName: aws.String(h.Store.TableName()),
			Key: map[string]types.AttributeValue{
				"PK": dynamo.S("PAYMENT#" + pi.ID),
				"SK": dynamo.S(sk),
			},
			UpdateExpression:          aws.String(updateExpr),
			ExpressionAttributeNames:  names,
			ExpressionAttributeValues: values,
		},
	}

	eventPut := types.TransactWriteItem{
		Put: &types.Put{
			TableName:           aws.String(h.Store.TableName()),
			Item:                eventItem,
			ConditionExpression: aws.String("attribute_not_exists(PK)"),
		},
	}

	if err := h.Store.TransactWrite(ctx, []types.TransactWriteItem{eventPut, paymentUpdate}); err != nil {
		if isConditionalCheckFailed(err) {
			h.Log.Info("evento_ja_processado", map[string]interface{}{"eventId": event.ID})
			return map[string]string{"status": "ok"}, nil
		}
		h.Log.Error("erro_processar_evento_sem_metadata", map[string]interface{}{"error": err.Error(), "eventId": event.ID, "paymentIntentId": pi.ID})
		return map[string]string{"status": "error"}, err
	}

	h.Log.Info("evento_processado_sem_metadata", map[string]interface{}{
		"eventId":         event.ID,
		"paymentIntentId": pi.ID,
		"status":          string(paymentStatus),
		"amount":          pi.Amount,
		"currency":        string(pi.Currency),
	})
	return map[string]string{"status": "ok"}, nil
}

func getStringAttr(item map[string]types.AttributeValue, key string) string {
	if val, ok := item[key]; ok {
		if s, ok := val.(*types.AttributeValueMemberS); ok {
			return s.Value
		}
	}
	return ""
}

func getNumberAttr(item map[string]types.AttributeValue, key string) string {
	if val, ok := item[key]; ok {
		if n, ok := val.(*types.AttributeValueMemberN); ok {
			return n.Value
		}
	}
	return ""
}

func parseInt64(value string) (int64, error) {
	if value == "" {
		return 0, errors.New("valor vazio")
	}
	var out int64
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, errors.New("valor invalido")
		}
		out = out*10 + int64(r-'0')
	}
	return out, nil
}

func intToString(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := false
	if value < 0 {
		negative = true
		value = -value
	}
	var digits [20]byte
	pos := len(digits)
	for value > 0 {
		pos--
		digits[pos] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		pos--
		digits[pos] = '-'
	}
	return string(digits[pos:])
}

func isConditionalCheckFailed(err error) bool {
	var txErr *types.TransactionCanceledException
	if errors.As(err, &txErr) {
		for _, reason := range txErr.CancellationReasons {
			if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" {
				return true
			}
		}
	}
	return false
}
