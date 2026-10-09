from enum import Enum

from pydantic import BaseModel, ConfigDict
from pydantic.alias_generators import to_camel


class CamelModel(BaseModel):
    """JSON uses camelCase field names, matching the other order-service implementations."""

    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)


class OrderRequest(CamelModel):
    order_id: str
    amount: float
    item: str
    quantity: int
    address: str


class OrderStatus(str, Enum):
    PENDING = "PENDING"
    PAYMENT_PROCESSING = "PAYMENT_PROCESSING"
    RESERVING_INVENTORY = "RESERVING_INVENTORY"
    AWAITING_GRACE_PERIOD = "AWAITING_GRACE_PERIOD"
    AWAITING_APPROVAL = "AWAITING_APPROVAL"
    DISPATCHING = "DISPATCHING"
    COMPLETED = "COMPLETED"
    CANCELLING = "CANCELLING"
    CANCELLED = "CANCELLED"
    FAILED = "FAILED"


class OrderResult(CamelModel):
    order_id: str
    status: OrderStatus
    message: str

    @classmethod
    def success(cls, order_id: str) -> "OrderResult":
        return cls(order_id=order_id, status=OrderStatus.COMPLETED, message="Order dispatched successfully.")

    @classmethod
    def cancelled(cls, order_id: str, reason: str) -> "OrderResult":
        return cls(order_id=order_id, status=OrderStatus.CANCELLED, message=f"Cancelled: {reason}")

    @classmethod
    def failed(cls, order_id: str, reason: str) -> "OrderResult":
        return cls(order_id=order_id, status=OrderStatus.FAILED, message=f"Failed: {reason}")
