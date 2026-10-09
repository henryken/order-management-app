import type { ErrorRequestHandler, RequestHandler } from 'express';
import { DownstreamServices, DownstreamServiceError, OutOfStockError } from './infra/downstream';
import type { OrderRequest } from './models';

/**
 * Processes an order with plain synchronous calls to the downstream services:
 * no Temporal, no retries, no compensation.
 */
export const createNaiveOrder =
  (services: DownstreamServices): RequestHandler =>
  async (req, res) => {
    const order = req.body as OrderRequest;
    await services.chargePayment(order.orderId, order.amount);
    await services.reserveInventory(order.item, order.quantity);
    await services.dispatchShipping(order.orderId, order.address);
    await services.sendNotification(order.orderId, 'DISPATCHED', 'Dispatched!');
    res.status(202).end();
  };

/** Turns a downstream failure into the HTTP status reported to the caller. */
export const downstreamErrorHandler: ErrorRequestHandler = (err, _req, res, next) => {
  if (err instanceof OutOfStockError) {
    res.status(400).json({ error: err.message });
  } else if (err instanceof DownstreamServiceError) {
    const status =
      err.status === undefined
        ? 503 // unreachable
        : err.status >= 500
          ? 502 // the service's own failure, not our caller's
          : err.status;
    res.status(status).json({ error: err.message });
  } else {
    next(err);
  }
};
