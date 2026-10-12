import { defineSearchAttributeKey, SearchAttributeType } from '@temporalio/common';
import {
  condition,
  defineQuery,
  defineSignal,
  proxyActivities,
  setCurrentDetails,
  setHandler,
  upsertSearchAttributes,
} from '@temporalio/workflow';
import { OrderResults } from '../models';
import type { OrderRequest, OrderResult, OrderStatus } from '../models';
import type { OrderActivities } from './activities';
import { Saga } from './saga';

const activities = proxyActivities<OrderActivities>({
  startToCloseTimeout: '5 seconds',
});

export const getStatusQuery = defineQuery<OrderStatus>('getStatus');
export const approveDispatchSignal = defineSignal<[string]>('approveDispatch');
export const cancelOrderSignal = defineSignal<[string]>('cancelOrder');

// How long an order can still be cancelled before it goes for approval.
const GRACE_PERIOD = '20 seconds';

// Custom search attribute, so orders can be listed and filtered by status in the Web UI,
// e.g. OrderStatus = "AWAITING_APPROVAL". It must be registered on the Temporal namespace.
const orderStatusKey = defineSearchAttributeKey('OrderStatus', SearchAttributeType.KEYWORD);

// Registered under the workflow type "OrderWorkflow", matching the Java implementation.
export async function OrderWorkflow(order: OrderRequest): Promise<OrderResult> {
  const saga = new Saga();

  let status: OrderStatus = 'PENDING';
  let isApproved = false;
  let approverEmail: string | undefined;
  let isCancelled = false;
  let cancellationReason = '';

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
  setHandler(cancelOrderSignal, (reason) => {
    isCancelled = true;
    cancellationReason = reason;
  });

  const handleCancellation = async (): Promise<OrderResult> => {
    setStatus('CANCELLING');
    await saga.compensate();
    setStatus('CANCELLED');

    await activities.SendNotification(order.orderId, 'CANCELLED', `Order cancelled: ${cancellationReason}`);

    return OrderResults.cancelled(order.orderId, cancellationReason);
  };

  // Payment
  setStatus('PAYMENT_PROCESSING');
  await activities.ChargePayment(order.orderId, order.amount);
  saga.addCompensation(() => activities.RefundPayment(order.orderId, order.amount));

  // Inventory
  setStatus('RESERVING_INVENTORY');
  await activities.ReserveInventory(order.item, order.quantity);
  saga.addCompensation(() => activities.ReleaseInventory(order.item, order.quantity));

  // Grace Period for cancellation
  setStatus('AWAITING_GRACE_PERIOD');
  const cancelledDuringGrace = await condition(() => isCancelled, GRACE_PERIOD);

  if (cancelledDuringGrace) {
    return handleCancellation();
  }

  // Human in the Loop: Awaiting Dispatch Approval
  setStatus('AWAITING_APPROVAL');
  await activities.SendNotification(order.orderId, 'APPROVAL_REQUIRED', 'Order awaiting manual dispatch approval.');

  // Waits durably without consuming threads or resources until approved (or cancelled)
  // The current details are shown in the Web UI while the workflow waits.
  setCurrentDetails(
    `**Awaiting dispatch approval** for order \`${order.orderId}\`.\n\n` +
      'Send the `approveDispatch` signal to continue, or `cancelOrder` to cancel.',
  );
  await condition(() => isApproved || isCancelled);
  setCurrentDetails('');

  if (isCancelled) {
    return handleCancellation();
  }

  // Dispatch
  setStatus('DISPATCHING');
  await activities.DispatchShipping(order.orderId, order.address);

  // Notification
  await activities.SendNotification(order.orderId, 'DISPATCHED', 'Dispatched!');

  setStatus('COMPLETED');

  return OrderResults.success(order.orderId);
}
