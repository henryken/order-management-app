using System.Net;

namespace OrderService.Infra;

/// <summary>A failed call to a downstream service: a non-2xx response or a connectivity problem.</summary>
public sealed class DownstreamServiceException(string message, HttpStatusCode? status = null, Exception? inner = null)
    : Exception(message, inner)
{
    public HttpStatusCode? Status { get; } = status;
}

/// <summary>The Inventory service rejected a reservation.</summary>
public sealed class OutOfStockException(string item) : Exception($"OUT_OF_STOCK: {item}");

/// <summary>HTTP clients for the downstream Payment, Inventory, Shipping and Notification services.</summary>
public sealed class DownstreamServices(HttpClient http, AppConfig config, ILogger<DownstreamServices> logger)
{
    private async Task PostAsync(string service, string baseUrl, string path, object body)
    {
        HttpResponseMessage response;
        try
        {
            response = await http.PostAsJsonAsync(baseUrl + path, body);
        }
        catch (Exception e) when (e is HttpRequestException or TaskCanceledException)
        {
            throw new DownstreamServiceException($"{service} is unreachable: {e.Message}", inner: e);
        }

        using (response)
        {
            if (!response.IsSuccessStatusCode)
            {
                throw new DownstreamServiceException(
                    $"{service} responded with {(int)response.StatusCode}", response.StatusCode);
            }
        }
    }

    public Task ChargePaymentAsync(string orderId, decimal amount)
    {
        logger.LogInformation("Charging {Amount} for order {OrderId}", amount, orderId);
        return PostAsync("payment-service", config.PaymentUrl, "/payment/charge", new { orderId, amount });
    }

    public Task RefundPaymentAsync(string orderId, decimal amount)
    {
        logger.LogInformation("Refunding {Amount} for order {OrderId}", amount, orderId);
        return PostAsync("payment-service", config.PaymentUrl, "/payment/refund", new { orderId, amount });
    }

    public async Task ReserveInventoryAsync(string item, int quantity)
    {
        logger.LogInformation("Reserving {Quantity} unit(s) of item {Item}", quantity, item);
        try
        {
            await PostAsync("inventory-service", config.InventoryUrl, "/inventory/reserve", new { item, quantity });
        }
        catch (DownstreamServiceException e) when (e.Status == HttpStatusCode.BadRequest)
        {
            throw new OutOfStockException(item);
        }
    }

    public Task ReleaseInventoryAsync(string item, int quantity)
    {
        logger.LogInformation("Releasing {Quantity} unit(s) of item {Item}", quantity, item);
        return PostAsync("inventory-service", config.InventoryUrl, "/inventory/release", new { item, quantity });
    }

    public Task DispatchShippingAsync(string orderId, string address)
    {
        logger.LogInformation("Dispatching order {OrderId} to {Address}", orderId, address);
        return PostAsync("shipping-service", config.ShippingUrl, "/shipping/dispatch", new { orderId, address });
    }

    public Task SendNotificationAsync(string orderId, string type, string message)
    {
        logger.LogInformation("Sending {Type} notification for order {OrderId}", type, orderId);
        return PostAsync("notification-service", config.NotificationUrl, "/notification/send",
            new { orderId, type, message });
    }
}
