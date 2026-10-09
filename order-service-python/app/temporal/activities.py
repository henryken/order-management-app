from temporalio import activity
from temporalio.exceptions import ApplicationError

from app.infra import DownstreamServices, OutOfStockError


class OrderActivities:
    """Activity names (ChargePayment, ReserveInventory, ...) match the Java implementation."""

    def __init__(self, services: DownstreamServices):
        self._services = services

    @activity.defn(name="ChargePayment")
    def charge_payment(self, order_id: str, amount: float) -> None:
        self._services.charge_payment(order_id, amount)

    @activity.defn(name="ReserveInventory")
    def reserve_inventory(self, item: str, quantity: int) -> None:
        try:
            self._services.reserve_inventory(item, quantity)
        except OutOfStockError as ex:
            raise ApplicationError(str(ex), type="ITEM_OUT_OF_STOCK", non_retryable=True) from ex

    @activity.defn(name="DispatchShipping")
    def dispatch_shipping(self, order_id: str, address: str) -> None:
        self._services.dispatch_shipping(order_id, address)

    @activity.defn(name="SendNotification")
    def send_notification(self, order_id: str, type_: str, message: str) -> None:
        self._services.send_notification(order_id, type_, message)
