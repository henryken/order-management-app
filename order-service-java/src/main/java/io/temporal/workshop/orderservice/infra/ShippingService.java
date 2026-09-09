package io.temporal.workshop.orderservice.infra;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestClient;
import org.springframework.web.client.RestClientException;
import java.util.Map;

@Service
public class ShippingService {
  private static final Logger log = LoggerFactory.getLogger(ShippingService.class);

  private final RestClient restClient;

  public ShippingService(@Value("${services.shipping-url}") String url) {
    this.restClient = RestClient.builder().baseUrl(url).build();
  }

  public void dispatch(String orderId, String address) {
    log.info("Dispatching shipment for order {} to {}", orderId, address);
    try {
      restClient.post()
          .uri("/shipping/dispatch")
          .body(Map.of("orderId", orderId, "address", address))
          .retrieve()
          .toBodilessEntity();
    } catch (RestClientException ex) {
      throw DownstreamCalls.translate("shipping-service", ex);
    }
  }
}
