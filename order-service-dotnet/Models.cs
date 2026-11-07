namespace OrderService;

public sealed record OrderRequest(string OrderId, decimal Amount, string Item, int Quantity, string Address);

// Serialized as the upper-snake-case names (AWAITING_APPROVAL, ...) shared with the other implementations.
public enum OrderStatus
{
    Pending,
    PaymentProcessing,
    ReservingInventory,
    AwaitingGracePeriod,
    AwaitingApproval,
    Dispatching,
    Completed,
    Cancelling,
    Cancelled,
    Failed,
}

public sealed record OrderResult(string OrderId, OrderStatus Status, string Message)
{
    public static OrderResult Success(string orderId) =>
        new(orderId, OrderStatus.Completed, "Order dispatched successfully.");

    public static OrderResult Cancelled(string orderId, string reason) =>
        new(orderId, OrderStatus.Cancelled, $"Cancelled: {reason}");

    public static OrderResult Failed(string orderId, string reason) =>
        new(orderId, OrderStatus.Failed, $"Failed: {reason}");
}
