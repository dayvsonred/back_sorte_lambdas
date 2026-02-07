package models

type DonationStatus string

type PaymentStatus string

const (
	DonationStatusCreated        DonationStatus = "CREATED"
	DonationStatusPendingPayment DonationStatus = "PENDING_PAYMENT"
	DonationStatusPaid           DonationStatus = "PAID"
	DonationStatusFailed         DonationStatus = "FAILED"
)

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusSucceeded PaymentStatus = "SUCCEEDED"
	PaymentStatusFailed    PaymentStatus = "FAILED"
)
