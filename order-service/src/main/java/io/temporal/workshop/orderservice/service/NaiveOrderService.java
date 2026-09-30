package io.temporal.workshop.orderservice.service;

import io.temporal.workshop.orderservice.infra.InventoryService;
import io.temporal.workshop.orderservice.infra.NotificationService;
import io.temporal.workshop.orderservice.infra.PaymentService;
import io.temporal.workshop.orderservice.infra.ShippingService;
import io.temporal.workshop.orderservice.model.OrderRequest;
import org.springframework.stereotype.Service;

@Service
public class NaiveOrderService {
  private final PaymentService paymentService;
  private final InventoryService inventoryService;
  private final ShippingService shippingService;
  private final NotificationService notificationService;

  public NaiveOrderService(PaymentService paymentService, InventoryService inventoryService,
      ShippingService shippingService, NotificationService notificationService) {
    this.paymentService = paymentService;
    this.inventoryService = inventoryService;
    this.shippingService = shippingService;
    this.notificationService = notificationService;
  }

  public void processOrder(OrderRequest order) {
    paymentService.charge(order.orderId(), order.amount());
    inventoryService.reserve(order.item(), order.quantity());
    shippingService.dispatch(order.orderId(), order.address());
    notificationService.send(order.orderId(), "DISPATCHED", "Dispatched!");
  }
}