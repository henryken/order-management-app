package io.temporal.workshop.orderservice.infra;

import org.springframework.http.HttpStatus;
import org.springframework.web.client.HttpStatusCodeException;
import org.springframework.web.client.RestClientException;

/**
 * Translates a {@link RestClientException} from a downstream infra service call into a
 * {@link DownstreamServiceException} carrying a sensible HTTP status to report back.
 */
final class DownstreamCalls {
  private DownstreamCalls() {}

  static DownstreamServiceException translate(String serviceName, RestClientException ex) {
    if (ex instanceof HttpStatusCodeException httpEx) {
      // A downstream 5xx is that service's own failure, not a bad request from us:
      // report it as a gateway failure rather than passing the raw 5xx through.
      HttpStatus status = httpEx.getStatusCode().is5xxServerError()
          ? HttpStatus.BAD_GATEWAY
          : HttpStatus.valueOf(httpEx.getStatusCode().value());

      return new DownstreamServiceException(status,
          "%s responded with %s".formatted(serviceName, httpEx.getStatusCode()), ex);
    }

    // Connection refused/reset, timeout, DNS failure, etc. - the service is unreachable.
    return new DownstreamServiceException(HttpStatus.SERVICE_UNAVAILABLE,
        "%s is unreachable: %s".formatted(serviceName, ex.getMessage()), ex);
  }
}
