package io.temporal.workshop.orderservice.temporal;

import io.temporal.activity.ActivityInterface;
import io.temporal.activity.ActivityMethod;

@ActivityInterface
public interface OrderActivities {
  @ActivityMethod
  void chargePayment(String orderId, double amount);

  @ActivityMethod
  void reserveInventory(String item, int quantity);

  @ActivityMethod
  void dispatchShipping(String orderId, String address);

  @ActivityMethod
  void sendNotification(String orderId, String type, String message);
}