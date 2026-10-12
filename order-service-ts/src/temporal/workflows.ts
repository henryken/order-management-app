import { defineSearchAttributeKey, SearchAttributeType } from '@temporalio/common';
import {
  condition,
  defineQuery,
  defineSignal,
  proxyActivities,
  setHandler,
  upsertSearchAttributes,
} from '@temporalio/workflow';
import { OrderResults } from '../models';
import type { OrderRequest, OrderResult, OrderStatus } from '../models';
import type { OrderActivities } from './activities';

const activities = proxyActivities<OrderActivities>({
  startToCloseTimeout: '5 seconds',
});

export const getStatusQuery = defineQuery<OrderStatus>('getStatus');
export const approveDispatchSignal = defineSignal<[string]>('approveDispatch');

// Custom search attribute, so orders can be listed and filtered by status in the Web UI,
// e.g. OrderStatus = "AWAITING_APPROVAL". It must be registered on the Temporal namespace.
const orderStatusKey = defineSearchAttributeKey('OrderStatus', SearchAttributeType.KEYWORD);

// Registered under the workflow type "OrderWorkflow", matching the Java implementation.
export async function OrderWorkflow(order: OrderRequest): Promise<OrderResult> {
  let status: OrderStatus = 'PENDING';
  let isApproved = false;
  let approverEmail: string | undefined;

  const setStatus = (newStatus: OrderStatus) => {
    status = newStatus;
    upsertSearchAttributes([{ key: orderStatusKey, value: newStatus }]);
  };
  setStatus('PENDING');

  setHandler(getStatusQuery, () => status);
  setHandler(approveDispatchSignal, (email) => {
    isApproved = true;
    approverEmail = email;
  });

  // Payment
  setStatus('PAYMENT_PROCESSING');
  await activities.ChargePayment(order.orderId, order.amount);

  // Inventory
  setStatus('RESERVING_INVENTORY');
  await activities.ReserveInventory(order.item, order.quantity);

  // Human in the Loop: Awaiting Dispatch Approval
  setStatus('AWAITING_APPROVAL');
  await activities.SendNotification(order.orderId, 'APPROVAL_REQUIRED', 'Order awaiting manual dispatch approval.');

  // Waits durably without consuming threads or resources until approved
  await condition(() => isApproved);

  // Dispatch
  setStatus('DISPATCHING');
  await activities.DispatchShipping(order.orderId, order.address);

  // Notification
  await activities.SendNotification(order.orderId, 'DISPATCHED', 'Dispatched!');

  setStatus('COMPLETED');

  return OrderResults.success(order.orderId);
}
