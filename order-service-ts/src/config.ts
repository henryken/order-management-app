const env = (key: string, fallback: string): string => process.env[key] || fallback;

export const config = {
  port: Number(env('PORT', '8080')),
  temporalAddress: env('TEMPORAL_ADDRESS', '127.0.0.1:7233'),
  temporalNamespace: env('TEMPORAL_NAMESPACE', 'default'),

  // The downstream services are reached through Toxiproxy (8501-8504), not directly.
  paymentUrl: env('SERVICES_PAYMENT_URL', 'http://localhost:8501'),
  inventoryUrl: env('SERVICES_INVENTORY_URL', 'http://localhost:8502'),
  shippingUrl: env('SERVICES_SHIPPING_URL', 'http://localhost:8503'),
  notificationUrl: env('SERVICES_NOTIFICATION_URL', 'http://localhost:8504'),
};

export type Config = typeof config;
