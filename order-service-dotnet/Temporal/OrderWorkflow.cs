using System.Text.Json;
using Temporalio.Common;
using Temporalio.Exceptions;
using Temporalio.Workflows;

namespace OrderService.Temporal;

// Registered under the workflow type "OrderWorkflow", matching the other implementations.
[Workflow("OrderWorkflow")]
public class OrderWorkflow
{
    // How long an order can still be cancelled before it goes for approval.
    private static readonly TimeSpan GracePeriod = TimeSpan.FromSeconds(20);

    private static readonly ActivityOptions Options = new() { StartToCloseTimeout = TimeSpan.FromSeconds(5) };

    // Custom search attribute, so orders can be listed and filtered by status in the Web UI,
    // e.g. OrderStatus = "AWAITING_APPROVAL". It must be registered on the Temporal namespace.
    private static readonly SearchAttributeKey<string> OrderStatusKey = SearchAttributeKey.CreateKeyword("OrderStatus");

    private OrderStatus status = OrderStatus.Pending;
    private readonly Saga saga = new();

    private bool isApproved;
    private string? approverEmail;
    private bool isCancelled;
    private string cancellationReason = "";

    [WorkflowQuery("getStatus")]
    public OrderStatus GetStatus() => status;

    [WorkflowSignal("approveDispatch")]
    public Task ApproveDispatchAsync(string? email = null)
    {
        // The email is optional so a signal sent without input (e.g. from the Web UI or the CLI)
        // still approves the order; otherwise the SDK silently drops it.
        isApproved = true;
        approverEmail = email ?? "manager@example.com";
        return Task.CompletedTask;
    }

    [WorkflowSignal("cancelOrder")]
    public Task CancelOrderAsync(string? reason = null)
    {
        // The reason is optional so a signal sent without input (e.g. from the Web UI or the CLI)
        // still cancels the order; otherwise the SDK silently drops it.
        isCancelled = true;
        cancellationReason = reason ?? "Customer changed mind";
        return Task.CompletedTask;
    }

    private void SetStatus(OrderStatus newStatus)
    {
        status = newStatus;
        // Upper-snake-case, e.g. AWAITING_APPROVAL, like the other implementations.
        Workflow.UpsertTypedSearchAttributes(
            OrderStatusKey.ValueSet(JsonNamingPolicy.SnakeCaseUpper.ConvertName(newStatus.ToString())));
    }

    private async Task<OrderResult> HandleCancellationAsync(OrderRequest order)
    {
        SetStatus(OrderStatus.Cancelling);
        await saga.CompensateAsync();
        SetStatus(OrderStatus.Cancelled);

        await Workflow.ExecuteActivityAsync((OrderActivities a) => a.SendNotificationAsync(order.OrderId, "CANCELLED", $"Order cancelled: {cancellationReason}"), Options);

        return OrderResult.Cancelled(order.OrderId, cancellationReason);
    }

    [WorkflowRun]
    public async Task<OrderResult> RunAsync(OrderRequest order)
    {
        SetStatus(OrderStatus.Pending);

        try
        {
            // Payment
            SetStatus(OrderStatus.PaymentProcessing);
            await Workflow.ExecuteActivityAsync((OrderActivities a) => a.ChargePaymentAsync(order.OrderId, order.Amount), Options);
            saga.AddCompensation(() => Workflow.ExecuteActivityAsync((OrderActivities a) => a.RefundPaymentAsync(order.OrderId, order.Amount), Options));

            // Inventory
            SetStatus(OrderStatus.ReservingInventory);
            await Workflow.ExecuteActivityAsync((OrderActivities a) => a.ReserveInventoryAsync(order.Item, order.Quantity), Options);
            saga.AddCompensation(() => Workflow.ExecuteActivityAsync((OrderActivities a) => a.ReleaseInventoryAsync(order.Item, order.Quantity), Options));

            // Grace Period for cancellation
            SetStatus(OrderStatus.AwaitingGracePeriod);
            var cancelledDuringGrace = await Workflow.WaitConditionAsync(() => isCancelled, GracePeriod);

            if (cancelledDuringGrace)
            {
                return await HandleCancellationAsync(order);
            }

            // Human in the Loop: Awaiting Dispatch Approval
            SetStatus(OrderStatus.AwaitingApproval);
            await Workflow.ExecuteActivityAsync((OrderActivities a) => a.SendNotificationAsync(order.OrderId, "APPROVAL_REQUIRED", "Order awaiting manual dispatch approval."), Options);

            // Waits durably without consuming threads or resources until approved (or cancelled)
            // The current details are shown in the Web UI while the workflow waits.
            Workflow.CurrentDetails =
                $"**Awaiting dispatch approval** for order `{order.OrderId}`.\n\n" +
                "Send the `approveDispatch` signal to continue, or `cancelOrder` to cancel.";
            await Workflow.WaitConditionAsync(() => isApproved || isCancelled);
            Workflow.CurrentDetails = "";

            if (isCancelled)
            {
                return await HandleCancellationAsync(order);
            }

            // Dispatch
            SetStatus(OrderStatus.Dispatching);
            await Workflow.ExecuteActivityAsync((OrderActivities a) => a.DispatchShippingAsync(order.OrderId, order.Address), Options);

            // Notification
            await Workflow.ExecuteActivityAsync((OrderActivities a) => a.SendNotificationAsync(order.OrderId, "DISPATCHED", "Dispatched!"), Options);

            SetStatus(OrderStatus.Completed);

            return OrderResult.Success(order.OrderId);
        }
        catch (ActivityFailureException e)
        {
            SetStatus(OrderStatus.Failed);
            await saga.CompensateAsync();

            // The message of the failure that caused the activity to fail
            var reason = e.InnerException?.Message ?? e.Message;
            await Workflow.ExecuteActivityAsync((OrderActivities a) => a.SendNotificationAsync(order.OrderId, "FAILED", $"Order failed: {reason}"), Options);

            throw;
        }
    }
}
