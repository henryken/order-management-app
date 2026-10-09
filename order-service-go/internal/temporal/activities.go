package temporal

import (
	"context"
	"errors"

	"go.temporal.io/sdk/temporal"

	"github.com/temporalio/order-management-app/order-service-go/internal/infra"
)

// Activities wraps the downstream services. Registering the struct exposes every exported
// method as an activity named after the method (ChargePayment, ReserveInventory, ...), which
// matches the activity names used by the Java implementation.
type Activities struct {
	Services *infra.Services
}

func (a *Activities) ChargePayment(ctx context.Context, orderID string, amount float64) error {
	return a.Services.ChargePayment(ctx, orderID, amount)
}

func (a *Activities) ReserveInventory(ctx context.Context, item string, quantity int) error {
	err := a.Services.ReserveInventory(ctx, item, quantity)
	if errors.Is(err, infra.ErrOutOfStock) {
		return temporal.NewNonRetryableApplicationError(err.Error(), "ITEM_OUT_OF_STOCK", err)
	}
	return err
}

func (a *Activities) DispatchShipping(ctx context.Context, orderID, address string) error {
	return a.Services.DispatchShipping(ctx, orderID, address)
}

func (a *Activities) SendNotification(ctx context.Context, orderID, notificationType, message string) error {
	return a.Services.SendNotification(ctx, orderID, notificationType, message)
}
