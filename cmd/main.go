package main

import (
	"app/config"
	"app/internal/handlers"
	"app/internal/repositories"
	"app/internal/services"
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	cfg := config.LoadConfig()
	
	paymentRepo := repositories.NewSeagm(cfg.PaymentConfig.Email, cfg.PaymentConfig.Password)
	pb := repositories.NewPocketBase(cfg.PocketBase.Address, cfg.PocketBase.Email, cfg.PocketBase.Password)

	exportService := services.NewExportService(paymentRepo, pb)
	verifyRepo := repositories.NewVerifyRepository()
	verifyService := services.NewVerifyService(pb, verifyRepo)

	exportHandler := handlers.NewExportHandler(exportService)
	schedulerHandler := handlers.NewSchedulerHandler(verifyService)
	schedulerHandler.StartVerifyPayment("payment")

	go func() {
		for {
			startListenPayment, stopListenPayment, recordChan, errChan := exportHandler.StartListeningOrder("payment")
			go func() {
				for {
					select {
					case record := <-recordChan:
						if err := exportHandler.ExportPayment("payment", record); err != nil {
							log.Println("Error exporting payment:", err)
						}
					case err := <-errChan:
						log.Println("Error listening for payment records:", err)
						stopListenPayment()
						return
					}
				}
			}()
			startListenPayment()
			log.Println("Started listening for new payment records...")
		}
	}()

	

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	fmt.Println("Shutting down gracefully, press Ctrl+C again to force")
	pb.Close()
	paymentRepo.Close()
	schedulerHandler.Stop()
	time.Sleep(7 * time.Second)
	fmt.Println("Server exiting")
}
