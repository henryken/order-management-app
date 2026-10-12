package temporal

import (
	"errors"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/temporalio/order-management-app/order-service-go/internal/model"
)

// Signal names match the Java implementation.
const (
	SignalApproveDispatch = "approveDispatch"
	SignalCancelOrder     = "cancelOrder"
)

// GracePeriod is how long an order can still be cancelled before it goes for approval.
const GracePeriod = 20 * time.Second

// orderStatusKey is a custom search attribute, so orders can be listed and filtered by status in
// the Web UI, e.g. OrderStatus = "AWAITING_APPROVAL". It must be registered on the Temporal namespace.
var orderStatusKey = temporal.NewSearchAttributeKeyKeyword("OrderStatus")

// OrderWorkflow is registered under the type name "OrderWorkflow" and answers the
// "getStatus" query and the "approveDispatch" and "cancelOrder" signals, matching the Java
// implementation.
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

	cancelled := false
	cancellationReason := ""
	cancelCh := workflow.GetSignalChannel(ctx, SignalCancelOrder)
	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			var reason string
			cancelCh.Receive(ctx, &reason)
			cancelled = true
			cancellationReason = reason
		}
	})

	var activities *Activities // only used to reference the activity methods
	s := &saga{}

	handleCancellation := func() (model.OrderResult, error) {
		setStatus(model.StatusCancelling)
		if err := s.compensate(ctx); err != nil {
			return model.OrderResult{}, err
		}
		setStatus(model.StatusCancelled)

		if err := workflow.ExecuteActivity(ctx, activities.SendNotification, order.OrderID, "CANCELLED", "Order cancelled: "+cancellationReason).Get(ctx, nil); err != nil {
			return model.OrderResult{}, err
		}
		return model.Cancelled(order.OrderID, cancellationReason), nil
	}

	process := func() (model.OrderResult, error) {
		// Payment
		setStatus(model.StatusPaymentProcessing)
		if err := workflow.ExecuteActivity(ctx, activities.ChargePayment, order.OrderID, order.Amount).Get(ctx, nil); err != nil {
			return model.OrderResult{}, err
		}
		s.addCompensation(func(ctx workflow.Context) error {
			return workflow.ExecuteActivity(ctx, activities.RefundPayment, order.OrderID, order.Amount).Get(ctx, nil)
		})

		// Inventory
		setStatus(model.StatusReservingInventory)
		if err := workflow.ExecuteActivity(ctx, activities.ReserveInventory, order.Item, order.Quantity).Get(ctx, nil); err != nil {
			return model.OrderResult{}, err
		}
		s.addCompensation(func(ctx workflow.Context) error {
			return workflow.ExecuteActivity(ctx, activities.ReleaseInventory, order.Item, order.Quantity).Get(ctx, nil)
		})

		// Grace Period for cancellation
		setStatus(model.StatusAwaitingGracePeriod)
		cancelledDuringGrace, err := workflow.AwaitWithTimeout(ctx, GracePeriod, func() bool { return cancelled })
		if err != nil {
			return model.OrderResult{}, err
		}
		if cancelledDuringGrace {
			return handleCancellation()
		}

		// Human in the Loop: Awaiting Dispatch Approval
		setStatus(model.StatusAwaitingApproval)
		if err := workflow.ExecuteActivity(ctx, activities.SendNotification, order.OrderID, "APPROVAL_REQUIRED", "Order awaiting manual dispatch approval.").Get(ctx, nil); err != nil {
			return model.OrderResult{}, err
		}

		// Waits durably without consuming threads or resources until approved (or cancelled)
		// The current details are shown in the Web UI while the workflow waits.
		workflow.SetCurrentDetails(ctx, "**Awaiting dispatch approval** for order `"+order.OrderID+
			"`.\n\nSend the `approveDispatch` signal to continue, or `cancelOrder` to cancel.")
		if err := workflow.Await(ctx, func() bool { return approved || cancelled }); err != nil {
			return model.OrderResult{}, err
		}
		workflow.SetCurrentDetails(ctx, "")
		if cancelled {
			return handleCancellation()
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

	result, err := process()

	var activityErr *temporal.ActivityError
	if errors.As(err, &activityErr) {
		setStatus(model.StatusFailed)
		if compErr := s.compensate(ctx); compErr != nil {
			return model.OrderResult{}, compErr
		}

		if notifyErr := workflow.ExecuteActivity(ctx, activities.SendNotification, order.OrderID, "FAILED", "Order failed: "+failureMessage(err)).Get(ctx, nil); notifyErr != nil {
			return model.OrderResult{}, notifyErr
		}

		return model.OrderResult{}, err
	}

	return result, err
}

// failureMessage returns the message of the failure that caused an activity to fail.
func failureMessage(err error) string {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		return appErr.Message()
	}
	return err.Error()
}
