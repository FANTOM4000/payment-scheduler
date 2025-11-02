package handlers

import (
	"app/internal/domains"
	"app/internal/services"
)

type ExportHandler interface {
	StartListeningOrder(collection string) (domains.Listening, domains.StopListening, chan domains.RecordHook[domains.PaymentRecord], chan error)
	ExportPayment(collection string, record domains.RecordHook[domains.PaymentRecord]) error
}

type exportHandler struct {
	ExportService services.ExportService
}

func NewExportHandler(exportService services.ExportService) ExportHandler {
	return &exportHandler{
		ExportService: exportService,
	}
}

func (h *exportHandler) StartListeningOrder(collection string) (domains.Listening, domains.StopListening, chan domains.RecordHook[domains.PaymentRecord], chan error) {
	Start, Stop, RecordChan, ErrChan := h.ExportService.ListeningOrder(collection)
	return Start, Stop, RecordChan, ErrChan
}

// func (h *exportHandler) ExportGGKeystorePayment(collection string, record domains.RecordHook[domains.PaymentRecord]) error {
// 	return h.ExportService.ExportGGKeyStorePayment(collection, record)
// }

func (h *exportHandler) ExportPayment(collection string, record domains.RecordHook[domains.PaymentRecord]) error {
	return h.ExportService.ExportPayment(collection, record)
}
