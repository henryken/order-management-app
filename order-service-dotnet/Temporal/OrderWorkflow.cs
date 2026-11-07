using Temporalio.Workflows;

namespace OrderService.Temporal;

// Registered under the workflow type "OrderWorkflow", matching the other implementations.
[Workflow("OrderWorkflow")]
public class OrderWorkflow
{
    private static readonly ActivityOptions Options = new() { StartToCloseTimeout = TimeSpan.FromSeconds(5) };

    private OrderStatus status = OrderStatus.Pending;

    [WorkflowQuery("getStatus")]
    public OrderStatus GetStatus() => status;

    [WorkflowRun]
    public async Task<OrderResult> RunAsync(OrderRequest order)
    {
        // Payment
        await Workflow.ExecuteActivityAsync((OrderActivities a) => a.ChargePaymentAsync(order.OrderId, order.Amount), Options);

        // Inventory
        await Workflow.ExecuteActivityAsync((OrderActivities a) => a.ReserveInventoryAsync(order.Item, order.Quantity), Options);

        // Dispatch
        await Workflow.ExecuteActivityAsync((OrderActivities a) => a.DispatchShippingAsync(order.OrderId, order.Address), Options);

        // Notification
        await Workflow.ExecuteActivityAsync((OrderActivities a) => a.SendNotificationAsync(order.OrderId, "DISPATCHED", "Dispatched!"), Options);

        status = OrderStatus.Completed;

        return OrderResult.Success(order.OrderId);
    }
}
