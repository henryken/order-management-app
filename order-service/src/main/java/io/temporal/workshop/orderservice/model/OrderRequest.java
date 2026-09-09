package io.temporal.workshop.orderservice.model;

public record OrderRequest(
    String orderId,
    double amount,
    String item,
    int quantity,
    String address
) {}