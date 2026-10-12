from datetime import timedelta

from temporalio import workflow
from temporalio.common import SearchAttributeKey

with workflow.unsafe.imports_passed_through():
    from app.models import OrderRequest, OrderResult, OrderStatus


# Custom search attribute, so orders can be listed and filtered by status in the Web UI,
# e.g. OrderStatus = "AWAITING_APPROVAL". It must be registered on the Temporal namespace.
ORDER_STATUS = SearchAttributeKey.for_keyword("OrderStatus")


@workflow.defn(name="OrderWorkflow")
class OrderWorkflow:
    def __init__(self) -> None:
        self._status = OrderStatus.PENDING
        self._approved = False
        self._approver_email: str | None = None

    @workflow.run
    async def process_order(self, order: OrderRequest) -> OrderResult:
        self._set_status(OrderStatus.PENDING)

        opts = {"start_to_close_timeout": timedelta(seconds=5)}

        # Payment
        self._set_status(OrderStatus.PAYMENT_PROCESSING)
        await workflow.execute_activity("ChargePayment", args=[order.order_id, order.amount], **opts)

        # Inventory
        self._set_status(OrderStatus.RESERVING_INVENTORY)
        await workflow.execute_activity("ReserveInventory", args=[order.item, order.quantity], **opts)

        # Human in the Loop: Awaiting Dispatch Approval
        self._set_status(OrderStatus.AWAITING_APPROVAL)
        await workflow.execute_activity(
            "SendNotification",
            args=[order.order_id, "APPROVAL_REQUIRED", "Order awaiting manual dispatch approval."],
            **opts,
        )

        # Waits durably without consuming threads or resources until approved
        await workflow.wait_condition(lambda: self._approved)

        # Dispatch
        self._set_status(OrderStatus.DISPATCHING)
        await workflow.execute_activity("DispatchShipping", args=[order.order_id, order.address], **opts)

        # Notification
        await workflow.execute_activity(
            "SendNotification", args=[order.order_id, "DISPATCHED", "Dispatched!"], **opts
        )

        self._set_status(OrderStatus.COMPLETED)

        return OrderResult.success(order.order_id)

    def _set_status(self, status: OrderStatus) -> None:
        self._status = status
        workflow.upsert_search_attributes([ORDER_STATUS.value_set(status.value)])

    @workflow.query(name="getStatus")
    def get_status(self) -> OrderStatus:
        return self._status

    @workflow.signal(name="approveDispatch")
    def approve_dispatch(self, approver_email: str | None = None) -> None:
        self._approved = True
        self._approver_email = approver_email
