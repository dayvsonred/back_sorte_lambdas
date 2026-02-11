package donation

import (
	"BACK_SORTE_GO/config"
	"BACK_SORTE_GO/internal/store"
	"BACK_SORTE_GO/internal/store/dynamo"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

const (
	emailEventTypeDonationCreated = "email-cadastro-doacao"
	emailEventTypeEmailVerify     = "email-validar-email-usuario"
)

type donationEmailEvent struct {
	Type           string `json:"type"`
	UserID         string `json:"user_id"`
	RecipientName  string `json:"recipient_name"`
	RecipientEmail string `json:"recipient_email"`
	DonationID     string `json:"donation_id"`
	DonationName   string `json:"donation_name"`
	DonationLink   string `json:"donation_link"`
	CreatedAt      string `json:"created_at"`
}

var (
	sqsClientOnce sync.Once
	sqsClient     *sqs.Client
	sqsClientErr  error
)

func getSQSClient(ctx context.Context) (*sqs.Client, error) {
	sqsClientOnce.Do(func() {
		region := config.GetAwsRegion()
		if region == "" {
			sqsClientErr = fmt.Errorf("AWS_REGION nao definido")
			return
		}
		cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
		if err != nil {
			sqsClientErr = fmt.Errorf("erro ao carregar config AWS para SQS: %w", err)
			return
		}
		sqsClient = sqs.NewFromConfig(cfg)
	})
	return sqsClient, sqsClientErr
}

func buildDonationPublicLink(nomeLink string) string {
	baseURL := strings.TrimRight(os.Getenv("APP_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "https://www.thepuregrace.com"
	}
	return fmt.Sprintf("%s/%s", baseURL, strings.TrimPrefix(nomeLink, "@"))
}

func publishDonationEmailEvent(ctx context.Context, event donationEmailEvent) error {
	queueURL := os.Getenv("EMAIL_EVENTS_QUEUE_URL")
	if queueURL == "" {
		return fmt.Errorf("EMAIL_EVENTS_QUEUE_URL nao definido")
	}

	client, err := getSQSClient(ctx)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("erro ao serializar evento de email: %w", err)
	}

	_, err = client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &queueURL,
		MessageBody: awsString(string(payload)),
	})
	if err != nil {
		return fmt.Errorf("erro ao enviar evento para SQS: %w", err)
	}
	return nil
}

func sendDonationCreatedEmailEvent(ctx context.Context, userID, recipientName, recipientEmail, donationID, donationName, donationLink string) error {
	event := donationEmailEvent{
		Type:           emailEventTypeDonationCreated,
		UserID:         userID,
		RecipientName:  recipientName,
		RecipientEmail: recipientEmail,
		DonationID:     donationID,
		DonationName:   donationName,
		DonationLink:   donationLink,
		CreatedAt:      time.Now().Format(time.RFC3339),
	}
	return publishDonationEmailEvent(ctx, event)
}

func sendEmailVerificationEvent(ctx context.Context, userID, recipientName, recipientEmail, donationID, donationName, donationLink string) error {
	event := donationEmailEvent{
		Type:           emailEventTypeEmailVerify,
		UserID:         userID,
		RecipientName:  recipientName,
		RecipientEmail: recipientEmail,
		DonationID:     donationID,
		DonationName:   donationName,
		DonationLink:   donationLink,
		CreatedAt:      time.Now().Format(time.RFC3339),
	}
	return publishDonationEmailEvent(ctx, event)
}

func lookupUserContact(ctx context.Context, storeDDB *dynamo.Store, userID string) (string, string, error) {
	item, err := storeDDB.GetItem(ctx, store.UserPK(userID), "PROFILE")
	if err != nil {
		return "", "", err
	}

	email := ""
	if raw, ok := item["email"]; ok {
		if v, ok := raw.(*types.AttributeValueMemberS); ok {
			email = strings.TrimSpace(v.Value)
		}
	}
	name := ""
	if raw, ok := item["name"]; ok {
		if v, ok := raw.(*types.AttributeValueMemberS); ok {
			name = strings.TrimSpace(v.Value)
		}
	}

	if email == "" {
		return "", "", fmt.Errorf("usuario %s sem email no PROFILE", userID)
	}
	return email, name, nil
}

func awsString(s string) *string {
	return &s
}
