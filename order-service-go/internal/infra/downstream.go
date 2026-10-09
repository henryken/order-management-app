// Package infra holds the HTTP clients for the downstream Payment, Inventory, Shipping and
// Notification services.
package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Config holds the base URLs of the downstream services.
type Config struct {
	PaymentURL      string
	InventoryURL    string
	ShippingURL     string
	NotificationURL string
}

// ErrOutOfStock is returned when the Inventory service rejects a reservation.
var ErrOutOfStock = errors.New("OUT_OF_STOCK")

// DownstreamError is a failed call to a downstream service: a non-2xx response or a
// connectivity problem (refused/reset connection, timeout).
type DownstreamError struct {
	Service string
	Status  int // 0 when the service was unreachable
	Err     error
}

func (e *DownstreamError) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("%s responded with %d", e.Service, e.Status)
	}
	return fmt.Sprintf("%s is unreachable: %v", e.Service, e.Err)
}

func (e *DownstreamError) Unwrap() error { return e.Err }

type Services struct {
	cfg  Config
	http *http.Client
}

func NewServices(cfg Config) *Services {
	return &Services{cfg: cfg, http: &http.Client{Timeout: 30 * time.Second}}
}

func (s *Services) post(ctx context.Context, service, baseURL, path string, body any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return &DownstreamError{Service: service, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &DownstreamError{Service: service, Status: resp.StatusCode,
			Err: fmt.Errorf("unexpected status %d", resp.StatusCode)}
	}
	return nil
}

func (s *Services) ChargePayment(ctx context.Context, orderID string, amount float64) error {
	slog.Info("Charging payment", "orderId", orderID, "amount", amount)
	return s.post(ctx, "payment-service", s.cfg.PaymentURL, "/payment/charge",
		map[string]any{"orderId": orderID, "amount": amount})
}

func (s *Services) RefundPayment(ctx context.Context, orderID string, amount float64) error {
	slog.Info("Refunding payment", "orderId", orderID, "amount", amount)
	return s.post(ctx, "payment-service", s.cfg.PaymentURL, "/payment/refund",
		map[string]any{"orderId": orderID, "amount": amount})
}

func (s *Services) ReserveInventory(ctx context.Context, item string, quantity int) error {
	slog.Info("Reserving inventory", "item", item, "quantity", quantity)
	err := s.post(ctx, "inventory-service", s.cfg.InventoryURL, "/inventory/reserve",
		map[string]any{"item": item, "quantity": quantity})
	var de *DownstreamError
	if errors.As(err, &de) && de.Status == http.StatusBadRequest {
		return fmt.Errorf("%w: %s", ErrOutOfStock, item)
	}
	return err
}

func (s *Services) ReleaseInventory(ctx context.Context, item string, quantity int) error {
	slog.Info("Releasing inventory", "item", item, "quantity", quantity)
	return s.post(ctx, "inventory-service", s.cfg.InventoryURL, "/inventory/release",
		map[string]any{"item": item, "quantity": quantity})
}

func (s *Services) DispatchShipping(ctx context.Context, orderID, address string) error {
	slog.Info("Dispatching shipment", "orderId", orderID, "address", address)
	return s.post(ctx, "shipping-service", s.cfg.ShippingURL, "/shipping/dispatch",
		map[string]any{"orderId": orderID, "address": address})
}

func (s *Services) SendNotification(ctx context.Context, orderID, notificationType, message string) error {
	slog.Info("Sending notification", "orderId", orderID, "type", notificationType)
	return s.post(ctx, "notification-service", s.cfg.NotificationURL, "/notification/send",
		map[string]any{"orderId": orderID, "type": notificationType, "message": message})
}
