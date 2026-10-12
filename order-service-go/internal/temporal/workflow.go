package temporal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/temporalio/order-management-app/order-service-go/internal/model"
)

// SignalApproveDispatch is the signal name, matching the Java implementation.
const SignalApproveDispatch = "approveDispatch"

// orderStatusKey is a custom search attribute, so orders can be listed and filtered by status in
// the Web UI, e.g. OrderStatus = "AWAITING_APPROVAL". It must be registered on the Temporal namespace.
var orderStatusKey = temporal.NewSearchAttributeKeyKeyword("OrderStatus")

// OrderWorkflow is registered under the type name "OrderWorkflow" and answers the
// "getStatus" query and the "approveDispatch" signal, matching the Java implementation.
func OrderWorkflow(ctx workflow.Context, order model.OrderRequest) (model.OrderResult, error) {
	status := model.StatusPending
	if err := workflow.SetQueryHandler(ctx, "getStatus", func() (model.OrderStatus, error) {
		return status, nil
	}); err != nil {
		return model.OrderResult{}, err
	}

	setStatus := func(s model.OrderStatus) {
		status = s
		// Upserting a keyword attribute can only fail on a local encoding error, which a string cannot cause.
		_ = workflow.UpsertTypedSearchAttributes(ctx, orderStatusKey.ValueSet(string(s)))
	}
	setStatus(model.StatusPending)

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	})

	approved := false
	approveCh := workflow.GetSignalChannel(ctx, SignalApproveDispatch)
	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			var approverEmail string
			approveCh.Receive(ctx, &approverEmail)
			approved = true
		}
	})

	var activities *Activities // only used to reference the activity methods

	// Payment
	setStatus(model.StatusPaymentProcessing)
	if err := workflow.ExecuteActivity(ctx, activities.ChargePayment, order.OrderID, order.Amount).Get(ctx, nil); err != nil {
		return model.OrderResult{}, err
	}

	// Inventory
	setStatus(model.StatusReservingInventory)
	if err := workflow.ExecuteActivity(ctx, activities.ReserveInventory, order.Item, order.Quantity).Get(ctx, nil); err != nil {
		return model.OrderResult{}, err
	}

	// Human in the Loop: Awaiting Dispatch Approval
	setStatus(model.StatusAwaitingApproval)
	if err := workflow.ExecuteActivity(ctx, activities.SendNotification, order.OrderID, "APPROVAL_REQUIRED", "Order awaiting manual dispatch approval.").Get(ctx, nil); err != nil {
		return model.OrderResult{}, err
	}

	// Waits durably without consuming threads or resources until approved
	if err := workflow.Await(ctx, func() bool { return approved }); err != nil {
		return model.OrderResult{}, err
	}

	// Dispatch
	setStatus(model.StatusDispatching)
	if err := workflow.ExecuteActivity(ctx, activities.DispatchShipping, order.OrderID, order.Address).Get(ctx, nil); err != nil {
		return model.OrderResult{}, err
	}

	// Notification
	if err := workflow.ExecuteActivity(ctx, activities.SendNotification, order.OrderID, "DISPATCHED", "Dispatched!").Get(ctx, nil); err != nil {
		return model.OrderResult{}, err
	}

	setStatus(model.StatusCompleted)

	return model.Success(order.OrderID), nil
}
