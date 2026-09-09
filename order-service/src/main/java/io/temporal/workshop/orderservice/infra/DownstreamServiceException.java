package io.temporal.workshop.orderservice.infra;

import org.springframework.http.HttpStatus;

/**
 * Wraps a failed call to a downstream infra service — a non-2xx response or a
 * connectivity problem (refused/reset connection, timeout) — with the HTTP status the
 * order-service should surface to its own caller.
 */
public class DownstreamServiceException extends RuntimeException {
  private final HttpStatus status;

  public DownstreamServiceException(HttpStatus status, String message, Throwable cause) {
    super(message, cause);
    this.status = status;
  }

  public HttpStatus getStatus() {
    return status;
  }
}
