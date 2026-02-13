package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/google/uuid"
)

const (
	emailTypeDonationCreated = "email-cadastro-doacao"
	emailTypeEmailVerify     = "email-validar-email-usuario"

	pendingPK = "EMAIL#PENDING"
	pendingSK = "TS#"
)

type emailEvent struct {
	Type           string `json:"type"`
	UserID         string `json:"user_id"`
	RecipientName  string `json:"recipient_name"`
	RecipientEmail string `json:"recipient_email"`
	DonationID     string `json:"donation_id"`
	DonationName   string `json:"donation_name"`
	DonationLink   string `json:"donation_link"`
	CreatedAt      string `json:"created_at"`
}

type appConfig struct {
	region          string
	tableName       string
	fromEmail       string
	fromName        string
	appBaseURL      string
	dailyEmailLimit int
	emailProvider   string
	brevoAPIKey     string
}

var (
	initOnce sync.Once
	cfg      appConfig
	ddb      *dynamodb.Client
	ses      *sesv2.Client
	initErr  error
)

func loadConfig(ctx context.Context) error {
	initOnce.Do(func() {
		cfg = appConfig{
			region:          env("AWS_REGION", "us-east-1"),
			tableName:       os.Getenv("DYNAMODB_TABLE"),
			fromEmail:       os.Getenv("SES_FROM_EMAIL"),
			fromName:        env("EMAIL_FROM_NAME", "The Pure Grace"),
			appBaseURL:      strings.TrimRight(env("APP_BASE_URL", "https://www.thepuregrace.com"), "/"),
			dailyEmailLimit: envInt("DAILY_EMAIL_LIMIT", 199),
			emailProvider:   strings.ToLower(env("EMAIL_PROVIDER", "ses")),
			brevoAPIKey:     strings.TrimSpace(os.Getenv("BREVO_API_KEY")),
		}
		if cfg.tableName == "" {
			initErr = fmt.Errorf("DYNAMODB_TABLE nao definido")
			return
		}
		if cfg.fromEmail == "" {
			initErr = fmt.Errorf("SES_FROM_EMAIL nao definido")
			return
		}
		if cfg.emailProvider == "brevo" && cfg.brevoAPIKey == "" {
			initErr = fmt.Errorf("BREVO_API_KEY nao definido para EMAIL_PROVIDER=brevo")
			return
		}

		awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.region))
		if err != nil {
			initErr = fmt.Errorf("erro ao carregar config AWS: %w", err)
			return
		}
		ddb = dynamodb.NewFromConfig(awsCfg)
		if cfg.emailProvider == "ses" {
			ses = sesv2.NewFromConfig(awsCfg)
		}
	})
	return initErr
}

func handler(ctx context.Context, raw json.RawMessage) error {
	if err := loadConfig(ctx); err != nil {
		return err
	}

	var sqsEvent events.SQSEvent
	if err := json.Unmarshal(raw, &sqsEvent); err == nil && len(sqsEvent.Records) > 0 && sqsEvent.Records[0].EventSource == "aws:sqs" {
		return processSQSEvent(ctx, sqsEvent)
	}

	var schedule events.CloudWatchEvent
	if err := json.Unmarshal(raw, &schedule); err == nil && schedule.Source == "aws.events" {
		return processPendingEmails(ctx)
	}

	log.Printf("evento nao tratado: %s", string(raw))
	return nil
}

func processSQSEvent(ctx context.Context, ev events.SQSEvent) error {
	for _, record := range ev.Records {
		var payload emailEvent
		if err := json.Unmarshal([]byte(record.Body), &payload); err != nil {
			log.Printf("payload invalido, ignorando mensagem %s: %v", record.MessageId, err)
			continue
		}

		sent, err := trySendEmail(ctx, payload)
		if err != nil {
			log.Printf("erro ao enviar email (%s), movendo para pendente: %v", payload.Type, err)
			if saveErr := savePendingEmail(ctx, payload); saveErr != nil {
				log.Printf("erro ao salvar pendente: %v", saveErr)
			}
			continue
		}
		if !sent {
			log.Printf("limite diario atingido, movendo para pendente: %s", payload.RecipientEmail)
			if saveErr := savePendingEmail(ctx, payload); saveErr != nil {
				log.Printf("erro ao salvar pendente: %v", saveErr)
			}
		}
	}
	return nil
}

func processPendingEmails(ctx context.Context) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var lastKey map[string]ddbtypes.AttributeValue

	for {
		out, err := ddb.Query(ctx, &dynamodb.QueryInput{
			TableName:              strPtr(cfg.tableName),
			KeyConditionExpression: strPtr("PK = :pk AND begins_with(SK, :sk)"),
			FilterExpression:       strPtr("#st = :pending AND next_attempt_at <= :now"),
			ExpressionAttributeNames: map[string]string{
				"#st": "status",
			},
			ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
				":pk":      &ddbtypes.AttributeValueMemberS{Value: pendingPK},
				":sk":      &ddbtypes.AttributeValueMemberS{Value: pendingSK},
				":pending": &ddbtypes.AttributeValueMemberS{Value: "PENDING"},
				":now":     &ddbtypes.AttributeValueMemberS{Value: now},
			},
			Limit:             int32Ptr(50),
			ExclusiveStartKey: lastKey,
		})
		if err != nil {
			return fmt.Errorf("erro ao buscar pendentes: %w", err)
		}

		for _, item := range out.Items {
			payloadStr := attrString(item["payload"])
			if payloadStr == "" {
				_ = markPendingAsFailed(ctx, item)
				continue
			}

			var payload emailEvent
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				log.Printf("payload pendente invalido: %v", err)
				_ = markPendingAsFailed(ctx, item)
				continue
			}

			sent, err := trySendEmail(ctx, payload)
			if err != nil {
				log.Printf("erro ao enviar pendente: %v", err)
				_ = reschedulePending(ctx, item)
				continue
			}
			if !sent {
				_ = reschedulePending(ctx, item)
				continue
			}

			if err := deletePending(ctx, item); err != nil {
				log.Printf("erro ao remover pendente enviado: %v", err)
			}
		}

		if len(out.LastEvaluatedKey) == 0 {
			break
		}
		lastKey = out.LastEvaluatedKey
	}

	return nil
}

func trySendEmail(ctx context.Context, payload emailEvent) (bool, error) {
	allowed, err := reserveDailyQuota(ctx)
	if err != nil {
		return false, err
	}
	if !allowed {
		return false, nil
	}

	subject, bodyText, err := buildEmailContent(ctx, payload)
	if err != nil {
		return false, err
	}

	err = sendEmailWithProvider(ctx, payload, subject, bodyText)
	if err != nil {
		return false, err
	}

	log.Printf("email enviado: type=%s to=%s donation_id=%s", payload.Type, payload.RecipientEmail, payload.DonationID)
	return true, nil
}

func sendEmailWithProvider(ctx context.Context, payload emailEvent, subject, bodyText string) error {
	switch cfg.emailProvider {
	case "ses":
		_, err := ses.SendEmail(ctx, &sesv2.SendEmailInput{
			FromEmailAddress: strPtr(cfg.fromEmail),
			Destination: &sestypes.Destination{
				ToAddresses: []string{payload.RecipientEmail},
			},
			Content: &sestypes.EmailContent{
				Simple: &sestypes.Message{
					Subject: &sestypes.Content{Data: strPtr(subject)},
					Body: &sestypes.Body{
						Text: &sestypes.Content{Data: strPtr(bodyText)},
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("erro SES SendEmail: %w", err)
		}
		return nil

	case "brevo":
		return sendEmailBrevo(ctx, payload, subject, bodyText)

	default:
		return fmt.Errorf("EMAIL_PROVIDER invalido: %s", cfg.emailProvider)
	}
}

func sendEmailBrevo(ctx context.Context, payload emailEvent, subject, bodyText string) error {
	type brevoContact struct {
		Email string `json:"email"`
		Name  string `json:"name,omitempty"`
	}
	type brevoSender struct {
		Email string `json:"email"`
		Name  string `json:"name,omitempty"`
	}
	type brevoSendEmailRequest struct {
		Sender      brevoSender    `json:"sender"`
		To          []brevoContact `json:"to"`
		Subject     string         `json:"subject"`
		TextContent string         `json:"textContent"`
	}

	reqBody := brevoSendEmailRequest{
		Sender: brevoSender{
			Email: cfg.fromEmail,
			Name:  cfg.fromName,
		},
		To: []brevoContact{{
			Email: payload.RecipientEmail,
			Name:  emptyIf(payload.RecipientName, "usuario"),
		}},
		Subject:     subject,
		TextContent: bodyText,
	}

	buf, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("erro ao serializar payload Brevo: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("erro ao criar request Brevo: %w", err)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("api-key", cfg.brevoAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao chamar Brevo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("erro Brevo status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

func buildEmailContent(ctx context.Context, payload emailEvent) (string, string, error) {
	link := strings.TrimSpace(payload.DonationLink)
	if link == "" {
		link = cfg.appBaseURL
	}
	whatsMsg := fmt.Sprintf("Apoie minha doacao %s: %s", payload.DonationName, link)
	whatsURL := "https://wa.me/?text=" + url.QueryEscape(whatsMsg)

	switch payload.Type {
	case emailTypeEmailVerify:
		token, err := createVerificationToken(ctx, payload)
		if err != nil {
			return "", "", err
		}
		confirmURL := fmt.Sprintf("%s/confirmar-email?token=%s", cfg.appBaseURL, token)
		subject := "Confirme seu e-mail - The Pure Grace"
		body := fmt.Sprintf(
			"Oi %s,\n\nSeu cadastro foi criado com sucesso.\n\nConfirme seu e-mail clicando no link:\n%s\n\nApos confirmar, voce pode acompanhar sua doacao no sistema.\n\nEquipe The Pure Grace",
			emptyIf(payload.RecipientName, "usuario"),
			confirmURL,
		)
		return subject, body, nil

	case emailTypeDonationCreated:
		subject := "Sua doacao foi criada com sucesso"
		body := fmt.Sprintf(
			"Oi %s,\n\nSua doacao \"%s\" foi criada com sucesso.\n\nLink da doacao:\n%s\n\nCompartilhe:\nWhatsApp: %s\nInstagram: compartilhe o link nos stories e na bio.\n\nImportante:\nFaca login no sistema e, no perfil, adicione seus dados bancarios para receber as doacoes.\n\nEquipe The Pure Grace",
			emptyIf(payload.RecipientName, "usuario"),
			emptyIf(payload.DonationName, "Minha doacao"),
			link,
			whatsURL,
		)
		return subject, body, nil
	default:
		return "", "", fmt.Errorf("tipo de email desconhecido: %s", payload.Type)
	}
}

func createVerificationToken(ctx context.Context, payload emailEvent) (string, error) {
	token := uuid.NewString()
	now := time.Now().UTC()
	exp := now.Add(48 * time.Hour)

	item := map[string]ddbtypes.AttributeValue{
		"PK":          &ddbtypes.AttributeValueMemberS{Value: "EMAIL#VERIFY#" + token},
		"SK":          &ddbtypes.AttributeValueMemberS{Value: "USER#" + payload.UserID},
		"user_id":     &ddbtypes.AttributeValueMemberS{Value: payload.UserID},
		"email":       &ddbtypes.AttributeValueMemberS{Value: payload.RecipientEmail},
		"donation_id": &ddbtypes.AttributeValueMemberS{Value: payload.DonationID},
		"used":        &ddbtypes.AttributeValueMemberBOOL{Value: false},
		"date_create": &ddbtypes.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		"expires_at":  &ddbtypes.AttributeValueMemberS{Value: exp.Format(time.RFC3339)},
	}

	_, err := ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: strPtr(cfg.tableName),
		Item:      item,
	})
	if err != nil {
		return "", fmt.Errorf("erro ao salvar token de validacao: %w", err)
	}
	return token, nil
}

func reserveDailyQuota(ctx context.Context) (bool, error) {
	now := time.Now().In(mustLocation("America/Sao_Paulo"))
	dateKey := now.Format("2006-01-02")
	pk := "EMAIL#QUOTA#" + dateKey
	sk := "COUNTER"

	_, err := ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: strPtr(cfg.tableName),
		Key: map[string]ddbtypes.AttributeValue{
			"PK": &ddbtypes.AttributeValueMemberS{Value: pk},
			"SK": &ddbtypes.AttributeValueMemberS{Value: sk},
		},
		UpdateExpression:    strPtr("SET send_count = if_not_exists(send_count, :zero) + :one, date_update = :now"),
		ConditionExpression: strPtr("attribute_not_exists(send_count) OR send_count < :limit"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":zero":  &ddbtypes.AttributeValueMemberN{Value: "0"},
			":one":   &ddbtypes.AttributeValueMemberN{Value: "1"},
			":limit": &ddbtypes.AttributeValueMemberN{Value: strconv.Itoa(cfg.dailyEmailLimit)},
			":now":   &ddbtypes.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		var condErr *ddbtypes.ConditionalCheckFailedException
		if strings.Contains(err.Error(), "ConditionalCheckFailedException") || (errors.As(err, &condErr) && condErr != nil) {
			return false, nil
		}
		return false, fmt.Errorf("erro ao reservar cota diaria: %w", err)
	}
	return true, nil
}

func savePendingEmail(ctx context.Context, payload emailEvent) error {
	serialized, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	next := nextRunAt7amSaoPaulo(now)

	item := map[string]ddbtypes.AttributeValue{
		"PK":              &ddbtypes.AttributeValueMemberS{Value: pendingPK},
		"SK":              &ddbtypes.AttributeValueMemberS{Value: fmt.Sprintf("TS#%d#%s", now.UnixMilli(), uuid.NewString())},
		"status":          &ddbtypes.AttributeValueMemberS{Value: "PENDING"},
		"payload":         &ddbtypes.AttributeValueMemberS{Value: string(serialized)},
		"attempts":        &ddbtypes.AttributeValueMemberN{Value: "0"},
		"next_attempt_at": &ddbtypes.AttributeValueMemberS{Value: next.Format(time.RFC3339)},
		"date_create":     &ddbtypes.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		"date_update":     &ddbtypes.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
	}

	_, err = ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: strPtr(cfg.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("erro ao salvar email pendente: %w", err)
	}
	return nil
}

func reschedulePending(ctx context.Context, item map[string]ddbtypes.AttributeValue) error {
	now := time.Now().UTC()
	next := nextRunAt7amSaoPaulo(now)
	_, err := ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: strPtr(cfg.tableName),
		Key: map[string]ddbtypes.AttributeValue{
			"PK": item["PK"],
			"SK": item["SK"],
		},
		UpdateExpression: strPtr("SET #st = :st, next_attempt_at = :next, date_update = :upd, attempts = if_not_exists(attempts, :zero) + :one"),
		ExpressionAttributeNames: map[string]string{
			"#st": "status",
		},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":st":   &ddbtypes.AttributeValueMemberS{Value: "PENDING"},
			":next": &ddbtypes.AttributeValueMemberS{Value: next.Format(time.RFC3339)},
			":upd":  &ddbtypes.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":zero": &ddbtypes.AttributeValueMemberN{Value: "0"},
			":one":  &ddbtypes.AttributeValueMemberN{Value: "1"},
		},
	})
	return err
}

func markPendingAsFailed(ctx context.Context, item map[string]ddbtypes.AttributeValue) error {
	now := time.Now().UTC()
	_, err := ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: strPtr(cfg.tableName),
		Key: map[string]ddbtypes.AttributeValue{
			"PK": item["PK"],
			"SK": item["SK"],
		},
		UpdateExpression: strPtr("SET #st = :st, date_update = :upd"),
		ExpressionAttributeNames: map[string]string{
			"#st": "status",
		},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":st":  &ddbtypes.AttributeValueMemberS{Value: "FAILED"},
			":upd": &ddbtypes.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
	})
	return err
}

func deletePending(ctx context.Context, item map[string]ddbtypes.AttributeValue) error {
	_, err := ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: strPtr(cfg.tableName),
		Key: map[string]ddbtypes.AttributeValue{
			"PK": item["PK"],
			"SK": item["SK"],
		},
	})
	return err
}

func nextRunAt7amSaoPaulo(now time.Time) time.Time {
	loc := mustLocation("America/Sao_Paulo")
	localNow := now.In(loc)
	nextLocal := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 7, 0, 0, 0, loc)
	if !localNow.Before(nextLocal) {
		nextLocal = nextLocal.Add(24 * time.Hour)
	}
	return nextLocal.UTC()
}

func env(key, fallback string) string {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func emptyIf(value, fallback string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return fallback
	}
	return v
}

func strPtr(s string) *string {
	return &s
}

func int32Ptr(v int32) *int32 {
	return &v
}

func attrString(v ddbtypes.AttributeValue) string {
	switch vv := v.(type) {
	case *ddbtypes.AttributeValueMemberS:
		return vv.Value
	default:
		return ""
	}
}

func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

func main() {
	lambda.Start(handler)
}
