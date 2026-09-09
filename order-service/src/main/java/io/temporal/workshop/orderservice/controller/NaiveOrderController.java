package io.temporal.workshop.orderservice.controller;

import io.temporal.workshop.orderservice.model.OrderRequest;
import io.temporal.workshop.orderservice.service.NaiveOrderService;
import jakarta.validation.Valid;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/orders")
public class NaiveOrderController {
  private final NaiveOrderService naiveOrderService;

  public NaiveOrderController(NaiveOrderService naiveOrderService) {
    this.naiveOrderService = naiveOrderService;
  }

  @PostMapping
  public ResponseEntity<Void> processOrder(@Valid @RequestBody OrderRequest order) {
    naiveOrderService.processOrder(order);
    return ResponseEntity.accepted().build();
  }
}
