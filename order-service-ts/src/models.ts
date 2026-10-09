export interface OrderRequest {
  orderId: string;
  amount: number;
  item: string;
  quantity: number;
  address: string;
}

export type OrderStatus =
  | 'PENDING'
  | 'PAYMENT_PROCESSING'
  | 'RESERVING_INVENTORY'
  | 'AWAITING_GRACE_PERIOD'
  | 'AWAITING_APPROVAL'
  | 'DISPATCHING'
  | 'COMPLETED'
  | 'CANCELLING'
  | 'CANCELLED'
  | 'FAILED';

export interface OrderResult {
  orderId: string;
  status: OrderStatus;
  message: string;
}

export const OrderResults = {
  success: (orderId: string): OrderResult => ({
    orderId,
    status: 'COMPLETED',
    message: 'Order dispatched successfully.',
  }),
  cancelled: (orderId: string, reason: string): OrderResult => ({
    orderId,
    status: 'CANCELLED',
    message: `Cancelled: ${reason}`,
  }),
  failed: (orderId: string, reason: string): OrderResult => ({
    orderId,
    status: 'FAILED',
    message: `Failed: ${reason}`,
  }),
};
