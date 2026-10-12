using System.Text.Json;
using Temporalio.Common;
using Temporalio.Workflows;

namespace OrderService.Temporal;

// Registered under the workflow type "OrderWorkflow", matching the other implementations.
[Workflow("OrderWorkflow")]
public class OrderWorkflow
{
    private static readonly ActivityOptions Options = new() { StartToCloseTimeout = TimeSpan.FromSeconds(5) };

    // Custom search attribute, so orders can be listed and filtered by status in the Web UI,
    // e.g. OrderStatus = "AWAITING_APPROVAL". It must be registered on the Temporal namespace.
    private static readonly SearchAttributeKey<string> OrderStatusKey = SearchAttributeKey.CreateKeyword("OrderStatus");

    private OrderStatus status = OrderStatus.Pending;
    private bool isApproved;
    private string? approverEmail;

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

    private void SetStatus(OrderStatus newStatus)
    {
        status = newStatus;
        // Upper-snake-case, e.g. AWAITING_APPROVAL, like the other implementations.
        Workflow.UpsertTypedSearchAttributes(
            OrderStatusKey.ValueSet(JsonNamingPolicy.SnakeCaseUpper.ConvertName(newStatus.ToString())));
    }

    [WorkflowRun]
    public async Task<OrderResult> RunAsync(OrderRequest order)
    {
        SetStatus(OrderStatus.Pending);

        // Payment
        SetStatus(OrderStatus.PaymentProcessing);
        await Workflow.ExecuteActivityAsync((OrderActivities a) => a.ChargePaymentAsync(order.OrderId, order.Amount), Options);

        // Inventory
        SetStatus(OrderStatus.ReservingInventory);
        await Workflow.ExecuteActivityAsync((OrderActivities a) => a.ReserveInventoryAsync(order.Item, order.Quantity), Options);

        // Human in the Loop: Awaiting Dispatch Approval
        SetStatus(OrderStatus.AwaitingApproval);
        await Workflow.ExecuteActivityAsync((OrderActivities a) => a.SendNotificationAsync(order.OrderId, "APPROVAL_REQUIRED", "Order awaiting manual dispatch approval."), Options);

        // Waits durably without consuming threads or resources until approved
        await Workflow.WaitConditionAsync(() => isApproved);

        // Dispatch
        SetStatus(OrderStatus.Dispatching);
        await Workflow.ExecuteActivityAsync((OrderActivities a) => a.DispatchShippingAsync(order.OrderId, order.Address), Options);

        // Notification
        await Workflow.ExecuteActivityAsync((OrderActivities a) => a.SendNotificationAsync(order.OrderId, "DISPATCHED", "Dispatched!"), Options);

        SetStatus(OrderStatus.Completed);

        return OrderResult.Success(order.OrderId);
    }
}
