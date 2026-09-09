package io.temporal.workshop.orderservice.controller;

import io.temporal.workshop.orderservice.infra.DownstreamServiceException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import java.util.Map;

/**
 * Turns exceptions raised while processing an order into a proper HTTP error response,
 * instead of Spring's default 500 with a raw stack trace.
 */
@RestControllerAdvice
public class GlobalExceptionHandler {
  private static final Logger log = LoggerFactory.getLogger(GlobalExceptionHandler.class);

  @ExceptionHandler(DownstreamServiceException.class)
  public ResponseEntity<Map<String, String>> handleDownstreamFailure(DownstreamServiceException ex) {
    log.warn("Downstream call failed: {}", ex.getMessage());
    return ResponseEntity.status(ex.getStatus()).body(Map.of("error", ex.getMessage()));
  }

  @ExceptionHandler(IllegalArgumentException.class)
  public ResponseEntity<Map<String, String>> handleBadRequest(IllegalArgumentException ex) {
    log.warn("Rejected order: {}", ex.getMessage());
    return ResponseEntity.status(HttpStatus.BAD_REQUEST).body(Map.of("error", ex.getMessage()));
  }
}
