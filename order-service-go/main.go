package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/temporalio/order-management-app/order-service-go/internal/api"
	"github.com/temporalio/order-management-app/order-service-go/internal/infra"
	ordertemporal "github.com/temporalio/order-management-app/order-service-go/internal/temporal"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	c, err := client.Dial(client.Options{
		HostPort:  env("TEMPORAL_ADDRESS", "127.0.0.1:7233"),
		Namespace: env("TEMPORAL_NAMESPACE", "default"),
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client:", err)
	}
	defer c.Close()

	// The downstream services are reached through Toxiproxy (8501-8504), not directly.
	services := infra.NewServices(infra.Config{
		PaymentURL:      env("SERVICES_PAYMENT_URL", "http://localhost:8501"),
		InventoryURL:    env("SERVICES_INVENTORY_URL", "http://localhost:8502"),
		ShippingURL:     env("SERVICES_SHIPPING_URL", "http://localhost:8503"),
		NotificationURL: env("SERVICES_NOTIFICATION_URL", "http://localhost:8504"),
	})

	w := worker.New(c, ordertemporal.TaskQueueOrderService, worker.Options{})
	w.RegisterWorkflow(ordertemporal.OrderWorkflow)
	w.RegisterActivity(&ordertemporal.Activities{Services: services})
	if err := w.Start(); err != nil {
		log.Fatalln("Unable to start worker:", err)
	}
	defer w.Stop()

	server := &http.Server{
		Addr:    ":" + env("PORT", "8080"),
		Handler: api.NewHandler(c, services).Routes(),
	}
	go func() {
		slog.Info("order-service-go listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalln("HTTP server failed:", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
