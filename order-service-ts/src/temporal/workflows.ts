import { defineQuery, proxyActivities, setHandler } from '@temporalio/workflow';
import { OrderResults } from '../models';
import type { OrderRequest, OrderResult, OrderStatus } from '../models';
import type { OrderActivities } from './activities';

const activities = proxyActivities<OrderActivities>({
  startToCloseTimeout: '5 seconds',
});

export const getStatusQuery = defineQuery<OrderStatus>('getStatus');

// Registered under the workflow type "OrderWorkflow", matching the Java implementation.
export async function OrderWorkflow(order: OrderRequest): Promise<OrderResult> {
  let status: OrderStatus = 'PENDING';
  setHandler(getStatusQuery, () => status);

  // Payment
  await activities.ChargePayment(order.orderId, order.amount);

  // Inventory
  await activities.ReserveInventory(order.item, order.quantity);

  // Dispatch
  await activities.DispatchShipping(order.orderId, order.address);

  // Notification
  await activities.SendNotification(order.orderId, 'DISPATCHED', 'Dispatched!');

  status = 'COMPLETED';

  return OrderResults.success(order.orderId);
}
