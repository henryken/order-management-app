package io.temporal.workshop.orderservice.infra;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestClient;
import org.springframework.web.client.RestClientException;
import java.util.Map;

@Service
public class NotificationService {
  private static final Logger log = LoggerFactory.getLogger(NotificationService.class);

  private final RestClient restClient;

  public NotificationService(@Value("${services.notification-url}") String url) {
    this.restClient = RestClient.builder().baseUrl(url).build();
  }

  public void send(String orderId, String type, String message) {
    log.info("Sending {} notification for order {}: {}", type, orderId, message);
    try {
      restClient.post()
          .uri("/notification/send")
          .body(Map.of("orderId", orderId, "type", type, "message", message))
          .retrieve()
          .toBodilessEntity();
    } catch (RestClientException ex) {
      throw DownstreamCalls.translate("notification-service", ex);
    }
  }
}
