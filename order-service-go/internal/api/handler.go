// Package api exposes the REST endpoints that start and inspect order workflows.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"go.temporal.io/sdk/client"

	"github.com/temporalio/order-management-app/order-service-go/internal/infra"
	"github.com/temporalio/order-management-app/order-service-go/internal/model"
	ordertemporal "github.com/temporalio/order-management-app/order-service-go/internal/temporal"
)

type Handler struct {
	services *infra.Services
	client   client.Client
}

func NewHandler(c client.Client, services *infra.Services) *Handler {
	return &Handler{client: c, services: services}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", h.createNaiveOrder)
	mux.HandleFunc("POST /temporal/orders", h.createOrder)
	mux.HandleFunc("GET /temporal/orders/{orderId}/status", h.getOrderStatus)
	return mux
}

func workflowID(orderID string) string { return "order-" + orderID }

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req model.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}

	_, err := h.client.ExecuteWorkflow(r.Context(), client.StartWorkflowOptions{
		ID:        workflowID(req.OrderID),
		TaskQueue: ordertemporal.TaskQueueOrderService,
	}, ordertemporal.OrderWorkflow, req)
	if err != nil {
		slog.Error("Failed to start workflow", "orderId", req.OrderID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"workflowId": workflowID(req.OrderID),
		"status":     "STARTED",
	})
}

func (h *Handler) getOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("orderId")

	resp, err := h.client.QueryWorkflow(r.Context(), workflowID(orderID), "", "getStatus")
	if err != nil {
		slog.Error("Failed to query workflow", "orderId", orderID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var status model.OrderStatus
	if err := resp.Get(&status); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"orderId": orderID, "status": status})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
