package services

import (
	"app/internal/domains"
	"app/internal/repositories"

	"github.com/shopspring/decimal"
)

type VerifyService interface {
	UpdateOrderStatus(collection string, id string, status string, message string) error
	AddCredit(userId string, amount decimal.Decimal) error
	GetPendingPayment() ([]domains.PaymentRecord, error)
	VerifyByUrl(url string) (bool, error)
}

func NewVerifyService(pocketBase repositories.PocketBase, verifyRepo repositories.VerifyRepository) VerifyService {
	return &verifyService{
		PocketBase: pocketBase,
		VerifyRepo: verifyRepo,
	}
}

type verifyService struct {
	PocketBase repositories.PocketBase
	VerifyRepo repositories.VerifyRepository
}

func (s *verifyService) VerifyByUrl(url string) (bool, error) {
	return s.VerifyRepo.VerifyByUrl(url)
}

func (s *verifyService) UpdateOrderStatus(collection string, id string, status string, message string) error {
	updateData := map[string]any{
		"status":  status,
		"message": message,
	}
	return s.PocketBase.UpdateRecord(collection, id, updateData)
}

func (s *verifyService) AddCredit(userId string, amount decimal.Decimal) error {
	createData := map[string]any{
		"userId":      userId,
		"amount":      amount,
		"description": "Deposit from payment",
		"type":        "ADD",
	}
	_, err := s.PocketBase.CreateRecord("creditTransactions", createData)
	return err
}

func (s *verifyService) GetPendingPayment() ([]domains.PaymentRecord, error) {
	records, err := s.PocketBase.GetPaymentRecordByFilter("payment", "status='user-paying' && paymentUrl != ''")
	if err != nil {
		return nil, err
	}
	return records, nil
}
