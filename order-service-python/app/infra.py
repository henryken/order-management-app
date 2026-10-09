"""HTTP clients for the downstream Payment, Inventory, Shipping and Notification services."""
import logging

import httpx

from app.config import Settings

log = logging.getLogger(__name__)


class DownstreamServiceError(Exception):
    """A failed call to a downstream service: a non-2xx response or a connectivity problem.

    `status` is the HTTP status of the response, or None when the service was unreachable.
    """

    def __init__(self, message: str, status: int | None = None):
        super().__init__(message)
        self.status = status


class OutOfStockError(Exception):
    """The Inventory service rejected a reservation."""


class DownstreamServices:
    def __init__(self, settings: Settings):
        self._settings = settings
        self._http = httpx.Client(timeout=30)

    def _post(self, service: str, url: str, path: str, body: dict) -> None:
        try:
            response = self._http.post(url + path, json=body)
            response.raise_for_status()
        except httpx.HTTPStatusError as ex:
            raise DownstreamServiceError(
                f"{service} responded with {ex.response.status_code}", ex.response.status_code
            ) from ex
        except httpx.HTTPError as ex:
            raise DownstreamServiceError(f"{service} is unreachable: {ex}") from ex

    def charge_payment(self, order_id: str, amount: float) -> None:
        log.info("Charging %s for order %s", amount, order_id)
        self._post("payment-service", self._settings.payment_url, "/payment/charge",
                   {"orderId": order_id, "amount": amount})

    def refund_payment(self, order_id: str, amount: float) -> None:
        log.info("Refunding %s for order %s", amount, order_id)
        self._post("payment-service", self._settings.payment_url, "/payment/refund",
                   {"orderId": order_id, "amount": amount})

    def reserve_inventory(self, item: str, quantity: int) -> None:
        log.info("Reserving %s unit(s) of item %s", quantity, item)
        try:
            self._post("inventory-service", self._settings.inventory_url, "/inventory/reserve",
                       {"item": item, "quantity": quantity})
        except DownstreamServiceError as ex:
            if ex.status == 400:
                raise OutOfStockError(f"OUT_OF_STOCK: {item}") from ex
            raise

    def release_inventory(self, item: str, quantity: int) -> None:
        log.info("Releasing %s unit(s) of item %s", quantity, item)
        self._post("inventory-service", self._settings.inventory_url, "/inventory/release",
                   {"item": item, "quantity": quantity})

    def dispatch_shipping(self, order_id: str, address: str) -> None:
        log.info("Dispatching order %s to %s", order_id, address)
        self._post("shipping-service", self._settings.shipping_url, "/shipping/dispatch",
                   {"orderId": order_id, "address": address})

    def send_notification(self, order_id: str, type_: str, message: str) -> None:
        log.info("Sending %s notification for order %s", type_, order_id)
        self._post("notification-service", self._settings.notification_url, "/notification/send",
                   {"orderId": order_id, "type": type_, "message": message})
