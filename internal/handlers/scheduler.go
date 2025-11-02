package handlers

import (
	"app/internal/domains"
	"app/internal/services"
	"log"

	"github.com/robfig/cron"
)

type SchedulerHandler interface {
	StartVerifyPayment(collection string) error
	Stop()
}

type schedulerHandler struct {
	cron          *cron.Cron
	verifyService services.VerifyService
	isRunning     bool
}

func NewSchedulerHandler(verifyService services.VerifyService) SchedulerHandler {
	return &schedulerHandler{
		cron:          cron.New(),
		verifyService: verifyService,
		isRunning:     true,
	}
}

func (h *schedulerHandler) StartVerifyPayment(collection string) error {
	h.cron.AddFunc("@every 1m", func() {
		pendingPayments, err := h.verifyService.GetPendingPayment()
		if err != nil {
			log.Println("Error fetching pending payments:", err)
			return
		}
		for _, payment := range pendingPayments {
			go func(payment domains.PaymentRecord) {
				success, err := h.verifyService.VerifyByUrl(payment.PaymentUrl)
				if err != nil {
					log.Println("Error verifying payment:", err)
					return
				}
				if success {
					err = h.verifyService.UpdateOrderStatus(collection, payment.Id, "success", "Payment verified from system")
					if err != nil {
						log.Println("Error updating order status:", err)
						return
					}
					err = h.verifyService.AddCredit(payment.UserId, payment.Amount)
					if err != nil {
						log.Println("Error adding credit:", err)
						return
					}
				} else {
					err = h.verifyService.UpdateOrderStatus(collection, payment.Id, "reject", "Payment verification failed")
					if err != nil {
						log.Println("Error updating order status:", err)
						return
					}
				}
			}(payment)
		}
	})
	h.cron.Start()
	return nil
}

func (h *schedulerHandler) Stop() {
	h.cron.Stop()
	h.isRunning = false
}
