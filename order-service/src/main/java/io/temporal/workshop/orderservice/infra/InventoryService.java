package io.temporal.workshop.orderservice.infra;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.web.client.HttpClientErrorException;
import org.springframework.web.client.RestClient;
import org.springframework.web.client.RestClientException;
import java.util.Map;

@Service
public class InventoryService {
  private static final Logger log = LoggerFactory.getLogger(InventoryService.class);

  private final RestClient restClient;

  public InventoryService(@Value("${services.inventory-url}") String url) {
    this.restClient = RestClient.builder().baseUrl(url).build();
  }

  public void reserve(String item, int quantity) {
    log.info("Reserving {} unit(s) of item {}", quantity, item);
    try {
      restClient.post()
          .uri("/inventory/reserve")
          .body(Map.of("item", item, "quantity", quantity))
          .retrieve()
          .toBodilessEntity();
    } catch (HttpClientErrorException.BadRequest ex) {
      throw new IllegalArgumentException("OUT_OF_STOCK: " + item);
    } catch (RestClientException ex) {
      throw DownstreamCalls.translate("inventory-service", ex);
    }
  }

  public void release(String item, int quantity) {
    log.info("Releasing {} unit(s) of item {}", quantity, item);
    try {
      restClient.post()
          .uri("/inventory/release")
          .body(Map.of("item", item, "quantity", quantity))
          .retrieve()
          .toBodilessEntity();
    } catch (RestClientException ex) {
      throw DownstreamCalls.translate("inventory-service", ex);
    }
  }
}
