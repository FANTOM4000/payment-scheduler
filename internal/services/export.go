package services

import (
	"app/internal/domains"
	"app/internal/ports"
	"app/internal/repositories"
	"fmt"
)

type exportService struct {
	PaymentRepo ports.PaymentRepository
	Pocketbase  repositories.PocketBase
}

type ExportService interface {
	ListeningOrder(collection string) (domains.Listening, domains.StopListening, chan domains.RecordHook[domains.PaymentRecord], chan error)

	ExportPayment(collection string, record domains.RecordHook[domains.PaymentRecord]) error
}

func NewExportService(paymentRepo ports.PaymentRepository, pb repositories.PocketBase) ExportService {
	return &exportService{
		PaymentRepo: paymentRepo,
		Pocketbase:  pb,
	}
}

func (s *exportService) ListeningOrder(collection string) (domains.Listening, domains.StopListening, chan domains.RecordHook[domains.PaymentRecord], chan error) {
	Start, Stop, RecordChan, ErrChan := s.Pocketbase.Subscribe(collection, domains.PaymentRecord{})

	return Start, Stop, RecordChan, ErrChan
}

func (s *exportService) ExportPayment(collection string, record domains.RecordHook[domains.PaymentRecord]) error {
	if record.Action != "create" {
		return nil
	}
	s.Pocketbase.UpdateRecord(collection, record.Record.Id, map[string]any{"status": "system-preparing"})
	fmt.Printf("New payment record created: %+v\n", record)
	s.Pocketbase.UpdateRecord(collection, record.Record.Id, map[string]any{"progress": 10})

	paymentInstance, err := s.PaymentRepo.NewPayment(record.Record.Id)
	if err != nil {
		s.Pocketbase.UpdateRecord(collection, record.Record.Id, map[string]any{"status": "reject", "message": fmt.Sprintf("Failed to create LapakGaming payment: %v", err), "progress": 100})
		fmt.Println("Failed to create LapakGaming payment:", err)
		return err
	}
	defer paymentInstance.Close()

	s.Pocketbase.UpdateRecord(collection, record.Record.Id, map[string]any{"progress": 40})
	if urlRedirect, qrCode, message, orderid, err := paymentInstance.SubmitPayment(record.Record.Id, domains.PromptPay, "", record.Record.Amount, func(progress uint) {
		fmt.Println("Payment progress:", progress)
		s.Pocketbase.UpdateRecord(collection, record.Record.Id, map[string]any{"progress": progress + 40})
	}); err != nil {
		s.Pocketbase.UpdateRecord(collection, record.Record.Id, map[string]any{"status": "reject", "message": fmt.Sprintf("Failed to submit payment: %v", err), "progress": 100})
		fmt.Println("Failed to submit payment:", err)
	} else {
		err = s.Pocketbase.UpdateRecord(collection, record.Record.Id, map[string]any{"status": "user-paying", "orderId": orderid, "qrCode": qrCode, "paymentUrl": urlRedirect, "message": message, "progress": 100})
		if err != nil {
			s.Pocketbase.UpdateRecord(collection, record.Record.Id, map[string]any{"status": "reject", "message": fmt.Sprintf("Failed to update record after payment submission: %v", err), "progress": 100})
			fmt.Println("Failed to update record after payment submission:", err)
			return err
		}
	}

	return nil
}
