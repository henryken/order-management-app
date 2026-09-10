package io.temporal.workshop.orderservice.controller;

import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowOptions;
import io.temporal.workshop.orderservice.model.OrderRequest;
import io.temporal.workshop.orderservice.model.OrderStatus;
import io.temporal.workshop.orderservice.temporal.OrderWorkflow;
import io.temporal.workshop.orderservice.temporal.TaskQueues;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Map;

@RestController
@RequestMapping("/temporal/orders")
public class TemporalOrderController {

  private final WorkflowClient workflowClient;

  public TemporalOrderController(WorkflowClient workflowClient) {
    this.workflowClient = workflowClient;
  }

  @PostMapping
  public ResponseEntity<Map<String, String>> createOrder(@RequestBody OrderRequest request) {
    WorkflowOptions options = WorkflowOptions.newBuilder()
        .setTaskQueue(TaskQueues.ORDER_SERVICE)
        .setWorkflowId("order-" + request.orderId())
        .build();

    OrderWorkflow workflow = workflowClient.newWorkflowStub(OrderWorkflow.class, options);
    WorkflowClient.start(workflow::processOrder, request);

    return ResponseEntity.accepted().body(Map.of(
        "workflowId", "order-" + request.orderId(),
        "status", "STARTED"
    ));
  }

  @PostMapping("/{orderId}/cancel")
  public ResponseEntity<Map<String, String>> cancelOrder(
      @PathVariable String orderId,
      @RequestParam(defaultValue = "Customer changed mind") String reason) {

    OrderWorkflow workflow = workflowClient.newWorkflowStub(OrderWorkflow.class, "order-" + orderId);
    workflow.cancelOrder(reason);

    return ResponseEntity.ok(Map.of("message", "Cancellation signal sent"));
  }

  @PostMapping("/{orderId}/approve")
  public ResponseEntity<Map<String, String>> approveOrder(
      @PathVariable String orderId,
      @RequestParam(defaultValue = "manager@example.com") String approverEmail) {

    OrderWorkflow workflow = workflowClient.newWorkflowStub(OrderWorkflow.class, "order-" + orderId);
    workflow.approveDispatch(approverEmail);

    return ResponseEntity.ok(Map.of("message", "Approval signal sent"));
  }

  @GetMapping("/{orderId}/status")
  public ResponseEntity<Map<String, Object>> getOrderStatus(@PathVariable String orderId) {
    OrderWorkflow workflow = workflowClient.newWorkflowStub(OrderWorkflow.class, "order-" + orderId);
    OrderStatus status = workflow.getStatus();

    return ResponseEntity.ok(Map.of("orderId", orderId, "status", status));
  }
}