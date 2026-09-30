package io.temporal.workshop.orderservice.temporal;

import io.temporal.workflow.QueryMethod;
import io.temporal.workflow.SignalMethod;
import io.temporal.workflow.WorkflowInterface;
import io.temporal.workflow.WorkflowMethod;
import io.temporal.workshop.orderservice.model.OrderRequest;
import io.temporal.workshop.orderservice.model.OrderResult;
import io.temporal.workshop.orderservice.model.OrderStatus;

@WorkflowInterface
public interface OrderWorkflow {

  @WorkflowMethod
  OrderResult processOrder(OrderRequest order);

  @QueryMethod
  OrderStatus getStatus();

}