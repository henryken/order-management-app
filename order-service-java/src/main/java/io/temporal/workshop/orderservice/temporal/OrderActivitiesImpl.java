package io.temporal.workshop.orderservice.temporal;

import io.temporal.failure.ApplicationFailure;
import io.temporal.spring.boot.ActivityImpl;
import io.temporal.workshop.orderservice.infra.InventoryService;
import io.temporal.workshop.orderservice.infra.NotificationService;
import io.temporal.workshop.orderservice.infra.PaymentService;
import io.temporal.workshop.orderservice.infra.ShippingService;
import org.springframework.stereotype.Component;

@Component
@ActivityImpl(taskQueues = TaskQueues.ORDER_SERVICE)
public class OrderActivitiesImpl implements OrderActivities {
  private final PaymentService paymentService;
  private final InventoryService inventoryService;
  private final ShippingService shippingService;
  private final NotificationService notificationService;

  public OrderActivitiesImpl(PaymentService paymentService, InventoryService inventoryService,
      ShippingService shippingService, NotificationService notificationService) {
    this.paymentService = paymentService;
    this.inventoryService = inventoryService;
    this.shippingService = shippingService;
    this.notificationService = notificationService;
  }

  @Override
  public void chargePayment(String orderId, double amount) {
    paymentService.charge(orderId, amount);
  }

  @Override
  public void refundPayment(String orderId, double amount) {
    paymentService.refund(orderId, amount);
  }

  @Override
  public void reserveInventory(String item, int quantity) {
    try {
      inventoryService.reserve(item, quantity);
    } catch (IllegalArgumentException e) {
      throw ApplicationFailure.newNonRetryableFailure(e.getMessage(), "ITEM_OUT_OF_STOCK");
    }
  }

  @Override
  public void releaseInventory(String item, int quantity) {
    inventoryService.release(item, quantity);
  }

  @Override
  public void dispatchShipping(String orderId, String address) {
    shippingService.dispatch(orderId, address);
  }

  @Override
  public void sendNotification(String orderId, String type, String message) {
    notificationService.send(orderId, type, message);
  }
}
