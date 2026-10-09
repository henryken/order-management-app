import express from 'express';
import type { Client } from '@temporalio/client';
import type { OrderRequest, OrderStatus } from './models';
import { ORDER_SERVICE } from './temporal/taskQueues';
import { createNaiveOrder, downstreamErrorHandler } from './naive';
import type { DownstreamServices } from './infra/downstream';
import { OrderWorkflow, getStatusQuery } from './temporal/workflows';

const workflowId = (orderId: string) => `order-${orderId}`;

export function createApp(client: Client, services: DownstreamServices) {
  const app = express();
  app.use(express.json());

  app.post('/orders', createNaiveOrder(services));

  app.post('/temporal/orders', async (req, res) => {
    const order = req.body as OrderRequest;
    await client.workflow.start(OrderWorkflow, {
      taskQueue: ORDER_SERVICE,
      workflowId: workflowId(order.orderId),
      args: [order],
    });
    res.status(202).json({ workflowId: workflowId(order.orderId), status: 'STARTED' });
  });

  app.get('/temporal/orders/:orderId/status', async (req, res) => {
    const { orderId } = req.params;
    const status: OrderStatus = await client.workflow.getHandle(workflowId(orderId)).query(getStatusQuery);
    res.json({ orderId, status });
  });

  app.use(downstreamErrorHandler);

  return app;
}
