package model

// OrderRequest is the payload that starts an order workflow.
type OrderRequest struct {
	OrderID  string  `json:"orderId"`
	Amount   float64 `json:"amount"`
	Item     string  `json:"item"`
	Quantity int     `json:"quantity"`
	Address  string  `json:"address"`
}

// OrderStatus is serialized as a plain string, matching the other order-service implementations.
type OrderStatus string

const (
	StatusPending             OrderStatus = "PENDING"
	StatusPaymentProcessing   OrderStatus = "PAYMENT_PROCESSING"
	StatusReservingInventory  OrderStatus = "RESERVING_INVENTORY"
	StatusAwaitingGracePeriod OrderStatus = "AWAITING_GRACE_PERIOD"
	StatusAwaitingApproval    OrderStatus = "AWAITING_APPROVAL"
	StatusDispatching         OrderStatus = "DISPATCHING"
	StatusCompleted           OrderStatus = "COMPLETED"
	StatusCancelling          OrderStatus = "CANCELLING"
	StatusCancelled           OrderStatus = "CANCELLED"
	StatusFailed              OrderStatus = "FAILED"
)

type OrderResult struct {
	OrderID string      `json:"orderId"`
	Status  OrderStatus `json:"status"`
	Message string      `json:"message"`
}

func Success(orderID string) OrderResult {
	return OrderResult{OrderID: orderID, Status: StatusCompleted, Message: "Order dispatched successfully."}
}

func Cancelled(orderID, reason string) OrderResult {
	return OrderResult{OrderID: orderID, Status: StatusCancelled, Message: "Cancelled: " + reason}
}

func Failed(orderID, reason string) OrderResult {
	return OrderResult{OrderID: orderID, Status: StatusFailed, Message: "Failed: " + reason}
}
