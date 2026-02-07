package stripeclient

import (
	"context"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/paymentintent"
	"github.com/stripe/stripe-go/v78/webhook"
)

type Client struct{}

func New(secretKey string) *Client {
	stripe.Key = secretKey
	return &Client{}
}

func (c *Client) CreatePaymentIntent(ctx context.Context, amount int64, currency, donationId, campaignId string) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
		Metadata: map[string]string{
			"donationId": donationId,
			"campaignId": campaignId,
		},
	}
	params.Context = ctx
	return paymentintent.New(params)
}

func (c *Client) CreateCheckoutSession(
	ctx context.Context,
	amount int64,
	currency,
	donationId,
	campaignId,
	donorName,
	donorEmail,
	successURL,
	cancelURL string,
) (*stripe.CheckoutSession, error) {
	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		CustomerEmail:     stripe.String(donorEmail),
		ClientReferenceID: stripe.String(donationId),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(currency),
					UnitAmount: stripe.Int64(amount),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Doacao"),
					},
				},
			},
		},
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"donationId": donationId,
				"campaignId": campaignId,
				"donorName":  donorName,
			},
		},
		Metadata: map[string]string{
			"donationId": donationId,
			"campaignId": campaignId,
		},
	}
	params.Context = ctx
	return session.New(params)
}

func (c *Client) VerifyWebhook(payload []byte, signature, secret string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, signature, secret)
}
