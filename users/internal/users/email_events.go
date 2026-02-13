package users

import (
	"BACK_SORTE_GO/config"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

const userEmailEventTypeEmailVerify = "email-validar-email-usuario"

type userEmailEvent struct {
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
	usersSQSClientOnce sync.Once
	usersSQSClient     *sqs.Client
	usersSQSClientErr  error
)

func getUsersSQSClient(ctx context.Context) (*sqs.Client, error) {
	usersSQSClientOnce.Do(func() {
		region := config.GetAwsRegion()
		if region == "" {
			usersSQSClientErr = fmt.Errorf("AWS_REGION nao definido")
			return
		}
		cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
		if err != nil {
			usersSQSClientErr = fmt.Errorf("erro ao carregar config AWS para SQS: %w", err)
			return
		}
		usersSQSClient = sqs.NewFromConfig(cfg)
	})
	return usersSQSClient, usersSQSClientErr
}

func publishUserEmailEvent(ctx context.Context, event userEmailEvent) error {
	queueURL := os.Getenv("EMAIL_EVENTS_QUEUE_URL")
	if queueURL == "" {
		return fmt.Errorf("EMAIL_EVENTS_QUEUE_URL nao definido")
	}

	client, err := getUsersSQSClient(ctx)
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

func awsString(s string) *string {
	return &s
}

func sendUserEmailVerificationEvent(ctx context.Context, userID, recipientName, recipientEmail string) error {
	event := userEmailEvent{
		Type:           userEmailEventTypeEmailVerify,
		UserID:         userID,
		RecipientName:  recipientName,
		RecipientEmail: recipientEmail,
		CreatedAt:      time.Now().Format(time.RFC3339),
	}
	return publishUserEmailEvent(ctx, event)
}
