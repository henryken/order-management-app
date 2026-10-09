package temporal

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"github.com/temporalio/order-management-app/order-service-go/internal/model"
)

// OrderWorkflow is registered under the type name "OrderWorkflow" and answers the
// "getStatus" query, matching the Java implementation.
func OrderWorkflow(ctx workflow.Context, order model.OrderRequest) (model.OrderResult, error) {
	status := model.StatusPending
	if err := workflow.SetQueryHandler(ctx, "getStatus", func() (model.OrderStatus, error) {
		return status, nil
	}); err != nil {
		return model.OrderResult{}, err
	}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	})

	var activities *Activities // only used to reference the activity methods

	// Payment
	if err := workflow.ExecuteActivity(ctx, activities.ChargePayment, order.OrderID, order.Amount).Get(ctx, nil); err != nil {
		return model.OrderResult{}, err
	}

	// Inventory
	if err := workflow.ExecuteActivity(ctx, activities.ReserveInventory, order.Item, order.Quantity).Get(ctx, nil); err != nil {
		return model.OrderResult{}, err
	}

	// Dispatch
	if err := workflow.ExecuteActivity(ctx, activities.DispatchShipping, order.OrderID, order.Address).Get(ctx, nil); err != nil {
		return model.OrderResult{}, err
	}

	// Notification
	if err := workflow.ExecuteActivity(ctx, activities.SendNotification, order.OrderID, "DISPATCHED", "Dispatched!").Get(ctx, nil); err != nil {
		return model.OrderResult{}, err
	}

	status = model.StatusCompleted

	return model.Success(order.OrderID), nil
}
