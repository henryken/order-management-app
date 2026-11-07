using System.Text.Json;
using System.Text.Json.Serialization;
using OrderService;
using OrderService.Infra;
using OrderService.Temporal;
using Temporalio.Client;
using Temporalio.Converters;

var config = AppConfig.FromEnvironment();

// camelCase JSON properties and upper-snake-case enum values, shared with the other implementations.
var json = new JsonSerializerOptions(JsonSerializerDefaults.Web)
{
    Converters = { new JsonStringEnumConverter(JsonNamingPolicy.SnakeCaseUpper) },
};

var builder = WebApplication.CreateBuilder(args);
builder.WebHost.UseUrls(Environment.GetEnvironmentVariable("ASPNETCORE_URLS") ?? $"http://0.0.0.0:{Environment.GetEnvironmentVariable("PORT") ?? "8080"}");
builder.Services.ConfigureHttpJsonOptions(o =>
{
    o.SerializerOptions.PropertyNamingPolicy = json.PropertyNamingPolicy;
    foreach (var c in json.Converters) o.SerializerOptions.Converters.Add(c);
});

var client = await TemporalClient.ConnectAsync(new(config.TemporalAddress)
{
    Namespace = config.TemporalNamespace,
    DataConverter = DataConverter.Default with { PayloadConverter = new DefaultPayloadConverter(json) },
});

builder.Services.AddSingleton(config);
builder.Services.AddSingleton<ITemporalClient>(client);
builder.Services.AddHttpClient<DownstreamServices>(c => c.Timeout = TimeSpan.FromSeconds(30));
builder.Services.AddSingleton<OrderActivities>();
builder.Services.AddHostedService<OrderWorker>();
builder.Services.AddExceptionHandler<DownstreamExceptionHandler>();
builder.Services.AddProblemDetails();

var app = builder.Build();
app.UseExceptionHandler();

string WorkflowId(string orderId) => $"order-{orderId}";

app.MapPost("/orders", Naive.CreateOrderAsync);

app.MapPost("/temporal/orders", async (OrderRequest order, ITemporalClient temporal) =>
{
    await temporal.StartWorkflowAsync(
        (OrderWorkflow wf) => wf.RunAsync(order),
        new(WorkflowId(order.OrderId), TaskQueues.OrderService));
    return Results.Accepted(value: new { workflowId = WorkflowId(order.OrderId), status = "STARTED" });
});

app.MapGet("/temporal/orders/{orderId}/status", async (string orderId, ITemporalClient temporal) =>
{
    var status = await temporal.GetWorkflowHandle<OrderWorkflow>(WorkflowId(orderId))
        .QueryAsync(wf => wf.GetStatus());
    return Results.Ok(new { orderId, status });
});

app.Run();
