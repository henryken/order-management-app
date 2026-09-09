package io.temporal.workshop.orderservice.infra;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestClient;
import org.springframework.web.client.RestClientException;
import java.util.Map;

@Service
public class PaymentService {
  private static final Logger log = LoggerFactory.getLogger(PaymentService.class);

  private final RestClient restClient;

  public PaymentService(@Value("${services.payment-url}") String url) {
    this.restClient = RestClient.builder().baseUrl(url).build();
  }

  public void charge(String orderId, double amount) {
    log.info("Charging {} for order {}", amount, orderId);
    try {
      restClient.post()
          .uri("/payment/charge")
          .body(Map.of("orderId", orderId, "amount", amount))
          .retrieve()
          .toBodilessEntity();
    } catch (RestClientException ex) {
      throw DownstreamCalls.translate("payment-service", ex);
    }
  }

  public void refund(String orderId, double amount) {
    log.info("Refunding {} for order {}", amount, orderId);
    try {
      restClient.post()
          .uri("/payment/refund")
          .body(Map.of("orderId", orderId, "amount", amount))
          .retrieve()
          .toBodilessEntity();
    } catch (RestClientException ex) {
      throw DownstreamCalls.translate("payment-service", ex);
    }
  }
}