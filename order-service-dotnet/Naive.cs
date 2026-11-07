using System.Net;
using Microsoft.AspNetCore.Diagnostics;
using OrderService.Infra;

namespace OrderService;

public static class Naive
{
    /// <summary>
    /// Processes an order with plain synchronous calls to the downstream services:
    /// no Temporal, no retries, no compensation.
    /// </summary>
    public static async Task<IResult> CreateOrderAsync(OrderRequest order, DownstreamServices services)
    {
        await services.ChargePaymentAsync(order.OrderId, order.Amount);
        await services.ReserveInventoryAsync(order.Item, order.Quantity);
        await services.DispatchShippingAsync(order.OrderId, order.Address);
        await services.SendNotificationAsync(order.OrderId, "DISPATCHED", "Dispatched!");
        return Results.Accepted();
    }
}

/// <summary>Turns a downstream failure into the HTTP status reported to the caller.</summary>
public sealed class DownstreamExceptionHandler : IExceptionHandler
{
    public async ValueTask<bool> TryHandleAsync(HttpContext context, Exception exception, CancellationToken ct)
    {
        int status;
        switch (exception)
        {
            case OutOfStockException:
                status = StatusCodes.Status400BadRequest;
                break;
            case DownstreamServiceException { Status: null }:
                status = StatusCodes.Status503ServiceUnavailable; // unreachable
                break;
            case DownstreamServiceException { Status: var s }:
                // A 5xx is the service's own failure, not our caller's.
                status = (int)s! >= 500 ? StatusCodes.Status502BadGateway : (int)s!;
                break;
            default:
                return false;
        }

        context.Response.StatusCode = status;
        await context.Response.WriteAsJsonAsync(new { error = exception.Message }, ct);
        return true;
    }
}
