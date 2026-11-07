using OrderService.Infra;
using Temporalio.Activities;

namespace OrderService.Temporal;

/// <summary>
/// Activity names are set explicitly (no "Async" suffix) so they match the names used by the
/// other implementations (ChargePayment, ReserveInventory, ...).
/// </summary>
public sealed class OrderActivities(DownstreamServices services)
{
    [Activity]
    public Task ChargePaymentAsync(string orderId, decimal amount) => services.ChargePaymentAsync(orderId, amount);

    [Activity]
    public Task ReserveInventoryAsync(string item, int quantity) => services.ReserveInventoryAsync(item, quantity);

    [Activity]
    public Task DispatchShippingAsync(string orderId, string address) => services.DispatchShippingAsync(orderId, address);

    [Activity]
    public Task SendNotificationAsync(string orderId, string type, string message) =>
        services.SendNotificationAsync(orderId, type, message);
}
