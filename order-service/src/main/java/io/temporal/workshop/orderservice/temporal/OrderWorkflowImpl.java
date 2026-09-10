package io.temporal.workshop.orderservice.temporal;

import io.temporal.activity.ActivityOptions;
import io.temporal.failure.ActivityFailure;
import io.temporal.spring.boot.WorkflowImpl;
import io.temporal.workflow.Saga;
import io.temporal.workflow.Workflow;
import io.temporal.workshop.orderservice.model.OrderRequest;
import io.temporal.workshop.orderservice.model.OrderResult;
import io.temporal.workshop.orderservice.model.OrderStatus;
import java.time.Duration;

@WorkflowImpl(taskQueues = TaskQueues.ORDER_SERVICE)
public class OrderWorkflowImpl implements OrderWorkflow {

  private OrderStatus status = OrderStatus.PENDING;

  private boolean isApproved = false;
  private String approverEmail;

  private boolean isCancelled = false;
  private String cancellationReason = "";

  private final ActivityOptions options = ActivityOptions.newBuilder()
      .setStartToCloseTimeout(Duration.ofSeconds(5))
      .build();

  private final OrderActivities activities = Workflow.newActivityStub(OrderActivities.class,
      options);

  @Override
  public OrderResult processOrder(OrderRequest order) {
    Saga saga = new Saga(new Saga.Options.Builder().setParallelCompensation(false).build());

    try {
      // Payment
      status = OrderStatus.PAYMENT_PROCESSING;
      activities.chargePayment(order.orderId(), order.amount());
      saga.addCompensation(() -> activities.refundPayment(order.orderId(), order.amount()));

      // Inventory
      status = OrderStatus.RESERVING_INVENTORY;
      activities.reserveInventory(order.item(), order.quantity());
      saga.addCompensation(() -> activities.releaseInventory(order.item(), order.quantity()));

      // Grace Period for cancellation
      status = OrderStatus.AWAITING_GRACE_PERIOD;
      boolean cancelledDuringGrace = Workflow.await(Duration.ofSeconds(20), () -> this.isCancelled);

      if (cancelledDuringGrace) {
        return handleCancellation(order, saga);
      }

      // Human in the Loop: Awaiting Dispatch Approval
      status = OrderStatus.AWAITING_APPROVAL;
      activities.sendNotification(order.orderId(), "APPROVAL_REQUIRED",
          "Order awaiting manual dispatch approval.");

      // Waits durably without consuming threads or resources until approved
      Workflow.await(() -> this.isApproved || this.isCancelled);

      if (this.isCancelled) {
        return handleCancellation(order, saga);
      }

      // Dispatch
      status = OrderStatus.DISPATCHING;
      activities.dispatchShipping(order.orderId(), order.address());

      // Notification
      activities.sendNotification(order.orderId(), "DISPATCHED", "Dispatched!");

      status = OrderStatus.COMPLETED;

      return OrderResult.success(order.orderId());
    } catch (ActivityFailure e) {
      status = OrderStatus.FAILED;
      saga.compensate();

      activities.sendNotification(order.orderId(), "FAILED", "Order failed: " + e.getMessage());

      throw e;
    }
  }

  private OrderResult handleCancellation(OrderRequest order, Saga saga) {
    status = OrderStatus.CANCELLING;
    saga.compensate();
    status = OrderStatus.CANCELLED;

    activities.sendNotification(order.orderId(), "CANCELLED", "Order cancelled: " + cancellationReason);

    return OrderResult.cancelled(order.orderId(), cancellationReason);
  }

  public OrderStatus getStatus() {
    return status;
  }

  @Override
  public void approveDispatch(String approverEmail) {
    this.isApproved = true;
    this.approverEmail = approverEmail;
  }

  @Override
  public void cancelOrder(String reason) {
    this.isCancelled = true;
    this.cancellationReason = reason;
  }

}