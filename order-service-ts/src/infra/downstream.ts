// HTTP clients for the downstream Payment, Inventory, Shipping and Notification services.
import type { Config } from '../config';

/** A failed call to a downstream service: a non-2xx response or a connectivity problem. */
export class DownstreamServiceError extends Error {
  constructor(
    message: string,
    readonly status?: number,
  ) {
    super(message);
    this.name = 'DownstreamServiceError';
  }
}

/** The Inventory service rejected a reservation. */
export class OutOfStockError extends Error {
  constructor(item: string) {
    super(`OUT_OF_STOCK: ${item}`);
    this.name = 'OutOfStockError';
  }
}

export class DownstreamServices {
  constructor(private readonly config: Config) {}

  private async post(service: string, baseUrl: string, path: string, body: object): Promise<void> {
    let response: Response;
    try {
      response = await fetch(baseUrl + path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal: AbortSignal.timeout(30_000),
      });
    } catch (err) {
      throw new DownstreamServiceError(`${service} is unreachable: ${(err as Error).message}`);
    }
    if (!response.ok) {
      throw new DownstreamServiceError(`${service} responded with ${response.status}`, response.status);
    }
  }

  chargePayment(orderId: string, amount: number): Promise<void> {
    console.log(`Charging ${amount} for order ${orderId}`);
    return this.post('payment-service', this.config.paymentUrl, '/payment/charge', { orderId, amount });
  }

  refundPayment(orderId: string, amount: number): Promise<void> {
    console.log(`Refunding ${amount} for order ${orderId}`);
    return this.post('payment-service', this.config.paymentUrl, '/payment/refund', { orderId, amount });
  }

  async reserveInventory(item: string, quantity: number): Promise<void> {
    console.log(`Reserving ${quantity} unit(s) of item ${item}`);
    try {
      await this.post('inventory-service', this.config.inventoryUrl, '/inventory/reserve', { item, quantity });
    } catch (err) {
      if (err instanceof DownstreamServiceError && err.status === 400) {
        throw new OutOfStockError(item);
      }
      throw err;
    }
  }

  releaseInventory(item: string, quantity: number): Promise<void> {
    console.log(`Releasing ${quantity} unit(s) of item ${item}`);
    return this.post('inventory-service', this.config.inventoryUrl, '/inventory/release', { item, quantity });
  }

  dispatchShipping(orderId: string, address: string): Promise<void> {
    console.log(`Dispatching order ${orderId} to ${address}`);
    return this.post('shipping-service', this.config.shippingUrl, '/shipping/dispatch', { orderId, address });
  }

  sendNotification(orderId: string, type: string, message: string): Promise<void> {
    console.log(`Sending ${type} notification for order ${orderId}`);
    return this.post('notification-service', this.config.notificationUrl, '/notification/send', {
      orderId,
      type,
      message,
    });
  }
}
