package io.temporal.workshop.orderservice.model;

public record OrderResult(
    String orderId,
    OrderStatus status,
    String message
) {
  public static OrderResult success(String orderId) {
    return new OrderResult(orderId, OrderStatus.COMPLETED, "Order dispatched successfully.");
  }
  public static OrderResult cancelled(String orderId, String reason) {
    return new OrderResult(orderId, OrderStatus.CANCELLED, "Cancelled: " + reason);
  }
  public static OrderResult failed(String orderId, String reason) {
    return new OrderResult(orderId, OrderStatus.FAILED, "Failed: " + reason);
  }
}