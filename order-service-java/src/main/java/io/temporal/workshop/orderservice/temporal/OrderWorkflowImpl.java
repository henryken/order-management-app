package io.temporal.workshop.orderservice.temporal;

import io.temporal.activity.ActivityOptions;
import io.temporal.spring.boot.WorkflowImpl;
import io.temporal.workflow.Workflow;

import io.temporal.workshop.orderservice.model.OrderRequest;
import io.temporal.workshop.orderservice.model.OrderResult;
import io.temporal.workshop.orderservice.model.OrderStatus;
import java.time.Duration;

@WorkflowImpl(taskQueues = TaskQueues.ORDER_SERVICE)
public class OrderWorkflowImpl implements OrderWorkflow {

  private OrderStatus status = OrderStatus.PENDING;

  private final ActivityOptions options = ActivityOptions.newBuilder()
      .setStartToCloseTimeout(Duration.ofSeconds(5))
      .build();

  private final OrderActivities activities = Workflow.newActivityStub(OrderActivities.class,
      options);

  @Override
  public OrderResult processOrder(OrderRequest order) {

    // Payment
    activities.chargePayment(order.orderId(), order.amount());

    // Inventory
    activities.reserveInventory(order.item(), order.quantity());

    // Dispatch
    activities.dispatchShipping(order.orderId(), order.address());

    // Notification
    activities.sendNotification(order.orderId(), "DISPATCHED", "Dispatched!");

    status = OrderStatus.COMPLETED;

    return OrderResult.success(order.orderId());
  }

  public OrderStatus getStatus() {
    return status;
  }

}