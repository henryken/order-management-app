using OrderService.Infra;
using Temporalio.Activities;
using Temporalio.Exceptions;

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
    public async Task ReserveInventoryAsync(string item, int quantity)
    {
        try
        {
            await services.ReserveInventoryAsync(item, quantity);
        }
        catch (OutOfStockException e)
        {
            throw new ApplicationFailureException(e.Message, errorType: "ITEM_OUT_OF_STOCK", nonRetryable: true);
        }
    }

    [Activity]
    public Task RefundPaymentAsync(string orderId, decimal amount) => services.RefundPaymentAsync(orderId, amount);

    [Activity]
    public Task ReleaseInventoryAsync(string item, int quantity) => services.ReleaseInventoryAsync(item, quantity);

    [Activity]
    public Task DispatchShippingAsync(string orderId, string address) => services.DispatchShippingAsync(orderId, address);

    [Activity]
    public Task SendNotificationAsync(string orderId, string type, string message) =>
        services.SendNotificationAsync(orderId, type, message);
}
