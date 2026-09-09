import express from 'express';

const createServiceApp = (serviceName) => {
  const app = express();
  app.use(express.json());
  app.use((req, res, next) => {
    console.log(`[${serviceName.toUpperCase()}] ${req.method} ${req.path} - Payload:`, req.body);
    next();
  });
  return app;
};

// 1. Payment Service (Port 8081)
const paymentApp = createServiceApp('payment');
paymentApp.post('/payment/charge', (req, res) => {
  const { orderId, amount } = req.body;
  console.log(`Charged $${amount} for order ${orderId}`);
  res.json({ status: 'CHARGED', transactionId: `txn_${Date.now()}` });
});
paymentApp.post('/payment/refund', (req, res) => {
  const { orderId, amount } = req.body;
  console.log(`Refunded $${amount} for order ${orderId}`);
  res.json({ status: 'REFUNDED' });
});
paymentApp.listen(8081, () => console.log('Payment running on 8081'));

// 2. Inventory Service (Port 8082)
const inventoryApp = createServiceApp('inventory');
inventoryApp.post('/inventory/reserve', (req, res) => {
  const { item, quantity } = req.body;
  if (item === 'OUT_OF_STOCK') {
    console.error(`Inventory unavailable for item: ${item}`);
    return res.status(400).json({ error: 'ITEM_OUT_OF_STOCK', message: `Item ${item} out of stock` });
  }
  console.log(`Reserved ${quantity}x ${item}`);
  res.json({ status: 'RESERVED' });
});
inventoryApp.post('/inventory/release', (req, res) => {
  const { item, quantity } = req.body;
  console.log(`Released hold on ${quantity}x ${item}`);
  res.json({ status: 'RELEASED' });
});
inventoryApp.listen(8082, () => console.log('Inventory running on 8082'));

// 3. Shipping Service (Port 8083)
const shippingApp = createServiceApp('shipping');
shippingApp.post('/shipping/dispatch', (req, res) => {
  const { orderId, address } = req.body;
  console.log(`Dispatched ${orderId} to ${address}`);
  res.json({ status: 'DISPATCHED', trackingNumber: `TRACK_${Date.now()}` });
});
shippingApp.listen(8083, () => console.log('Shipping running on 8083'));

// 4. Notification Service (Port 8084)
const notificationApp = createServiceApp('notification');
notificationApp.post('/notification/send', (req, res) => {
  const { orderId, type, message } = req.body;
  console.log(`Notification [${type}] for ${orderId}: ${message}`);
  res.json({ status: 'SENT' });
});
notificationApp.listen(8084, () => console.log('Notification running on 8084'));