import { DownstreamServices } from '../infra/downstream';

/**
 * Activity names are the object keys, so they are PascalCase on purpose: they match the
 * activity names used by the Java implementation (ChargePayment, ReserveInventory, ...).
 */
export const createActivities = (services: DownstreamServices) => ({
  async ChargePayment(orderId: string, amount: number): Promise<void> {
    await services.chargePayment(orderId, amount);
  },

  async ReserveInventory(item: string, quantity: number): Promise<void> {
    await services.reserveInventory(item, quantity);
  },

  async DispatchShipping(orderId: string, address: string): Promise<void> {
    await services.dispatchShipping(orderId, address);
  },

  async SendNotification(orderId: string, type: string, message: string): Promise<void> {
    await services.sendNotification(orderId, type, message);
  },
});

export type OrderActivities = ReturnType<typeof createActivities>;
