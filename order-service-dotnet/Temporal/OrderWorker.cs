using Temporalio.Client;
using Temporalio.Worker;

namespace OrderService.Temporal;

/// <summary>Runs the Temporal worker for the order-service task queue alongside the web server.</summary>
public sealed class OrderWorker(ITemporalClient client, OrderActivities activities) : BackgroundService
{
    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        using var worker = new TemporalWorker(
            client,
            new TemporalWorkerOptions(TaskQueues.OrderService)
                .AddActivity(activities.ChargePaymentAsync)
                .AddActivity(activities.ReserveInventoryAsync)
                .AddActivity(activities.DispatchShippingAsync)
                .AddActivity(activities.SendNotificationAsync)
                .AddWorkflow<OrderWorkflow>());

        await worker.ExecuteAsync(stoppingToken);
    }
}
