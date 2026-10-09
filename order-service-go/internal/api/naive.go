package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/temporalio/order-management-app/order-service-go/internal/infra"
	"github.com/temporalio/order-management-app/order-service-go/internal/model"
)

// createNaiveOrder processes an order with plain synchronous calls to the downstream
// services: no Temporal, no retries, no compensation.
func (h *Handler) createNaiveOrder(w http.ResponseWriter, r *http.Request) {
	var order model.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}

	ctx := r.Context()
	s := h.services
	var err error
	for _, step := range []func() error{
		func() error { return s.ChargePayment(ctx, order.OrderID, order.Amount) },
		func() error { return s.ReserveInventory(ctx, order.Item, order.Quantity) },
		func() error { return s.DispatchShipping(ctx, order.OrderID, order.Address) },
		func() error { return s.SendNotification(ctx, order.OrderID, "DISPATCHED", "Dispatched!") },
	} {
		if err = step(); err != nil {
			break
		}
	}
	if err != nil {
		status, message := translateError(err)
		slog.Warn("Order processing failed", "orderId", order.OrderID, "error", message)
		writeJSON(w, status, map[string]string{"error": message})
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// translateError maps a downstream failure to the HTTP status reported to the caller.
func translateError(err error) (int, string) {
	if errors.Is(err, infra.ErrOutOfStock) {
		return http.StatusBadRequest, err.Error()
	}
	var de *infra.DownstreamError
	if errors.As(err, &de) {
		switch {
		case de.Status == 0:
			return http.StatusServiceUnavailable, de.Error() // unreachable
		case de.Status >= 500:
			return http.StatusBadGateway, de.Error() // the service's own failure, not our caller's
		default:
			return de.Status, de.Error()
		}
	}
	return http.StatusInternalServerError, err.Error()
}
