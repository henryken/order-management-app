package io.temporal.workshop.orderservice.temporal;

import io.temporal.activity.ActivityOptions;
import io.temporal.common.SearchAttributeKey;
import io.temporal.spring.boot.WorkflowImpl;
import io.temporal.workflow.Saga;
import io.temporal.workflow.Workflow;
import io.temporal.workshop.orderservice.model.OrderRequest;
import io.temporal.workshop.orderservice.model.OrderResult;
import io.temporal.workshop.orderservice.model.OrderStatus;
import java.time.Duration;

@WorkflowImpl(taskQueues = TaskQueues.ORDER_SERVICE)
public class OrderWorkflowImpl implements OrderWorkflow {

  // Custom search attribute, so orders can be listed and filtered by status in the Web UI,
  // e.g. OrderStatus = "AWAITING_APPROVAL". It must be registered on the Temporal namespace.
  private static final SearchAttributeKey<String> ORDER_STATUS =
      SearchAttributeKey.forKeyword("OrderStatus");

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
    setStatus(OrderStatus.PENDING);

    Saga saga = new Saga(new Saga.Options.Builder().setParallelCompensation(false).build());

    // Payment
    setStatus(OrderStatus.PAYMENT_PROCESSING);
    activities.chargePayment(order.orderId(), order.amount());
    saga.addCompensation(() -> activities.refundPayment(order.orderId(), order.amount()));

    // Inventory
    setStatus(OrderStatus.RESERVING_INVENTORY);
    activities.reserveInventory(order.item(), order.quantity());
    saga.addCompensation(() -> activities.releaseInventory(order.item(), order.quantity()));

    // Grace Period for cancellation
    setStatus(OrderStatus.AWAITING_GRACE_PERIOD);
    boolean cancelledDuringGrace = Workflow.await(Duration.ofSeconds(20), () -> this.isCancelled);

    if (cancelledDuringGrace) {
      return handleCancellation(order, saga);
    }

    // Human in the Loop: Awaiting Dispatch Approval
    setStatus(OrderStatus.AWAITING_APPROVAL);
    activities.sendNotification(order.orderId(), "APPROVAL_REQUIRED",
        "Order awaiting manual dispatch approval.");

    // Waits durably without consuming threads or resources until approved
    // The current details are shown in the Web UI while the workflow waits.
    Workflow.setCurrentDetails("**Awaiting dispatch approval** for order `" + order.orderId()
        + "`.\n\nSend the `approveDispatch` signal to continue, or `cancelOrder` to cancel.");
    Workflow.await(() -> this.isApproved || this.isCancelled);
    Workflow.setCurrentDetails("");

    if (this.isCancelled) {
      return handleCancellation(order, saga);
    }

    // Dispatch
    setStatus(OrderStatus.DISPATCHING);
    activities.dispatchShipping(order.orderId(), order.address());

    // Notification
    activities.sendNotification(order.orderId(), "DISPATCHED", "Dispatched!");

    setStatus(OrderStatus.COMPLETED);

    return OrderResult.success(order.orderId());
  }

  private OrderResult handleCancellation(OrderRequest order, Saga saga) {
    setStatus(OrderStatus.CANCELLING);
    saga.compensate();
    setStatus(OrderStatus.CANCELLED);

    activities.sendNotification(order.orderId(), "CANCELLED", "Order cancelled: " + cancellationReason);

    return OrderResult.cancelled(order.orderId(), cancellationReason);
  }

  private void setStatus(OrderStatus newStatus) {
    status = newStatus;
    Workflow.upsertTypedSearchAttributes(ORDER_STATUS.valueSet(newStatus.name()));
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