namespace OrderService;

public sealed record AppConfig(
    string TemporalAddress,
    string TemporalNamespace,
    string PaymentUrl,
    string InventoryUrl,
    string ShippingUrl,
    string NotificationUrl)
{
    // Read from environment variables (the same ones the other order services use).
    // The downstream services are reached through Toxiproxy (8501-8504), not directly.
    public static AppConfig FromEnvironment() => new(
        TemporalAddress: Env("TEMPORAL_ADDRESS", "127.0.0.1:7233"),
        TemporalNamespace: Env("TEMPORAL_NAMESPACE", "default"),
        PaymentUrl: Env("SERVICES_PAYMENT_URL", "http://localhost:8501"),
        InventoryUrl: Env("SERVICES_INVENTORY_URL", "http://localhost:8502"),
        ShippingUrl: Env("SERVICES_SHIPPING_URL", "http://localhost:8503"),
        NotificationUrl: Env("SERVICES_NOTIFICATION_URL", "http://localhost:8504"));

    private static string Env(string key, string fallback) =>
        Environment.GetEnvironmentVariable(key) is { Length: > 0 } value ? value : fallback;
}
