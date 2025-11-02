package ports

import (
	"app/internal/domains"

	"github.com/shopspring/decimal"
)

type PaymentRepository interface {
	NewPayment(id string) (PaymentRepository, error)
	SubmitPayment(id string, method domains.PaymentMethod, phone string, amount decimal.Decimal, callBackProgress func(uint)) (urlRedirect, qrData, message, orderid string, err error)
	SubmitOtp(id string, otp string) (urlRedirect, qrData, message string, err error)
	Close()
}
