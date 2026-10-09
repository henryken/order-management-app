from datetime import timedelta

from temporalio import workflow

with workflow.unsafe.imports_passed_through():
    from app.models import OrderRequest, OrderResult, OrderStatus


@workflow.defn(name="OrderWorkflow")
class OrderWorkflow:
    def __init__(self) -> None:
        self._status = OrderStatus.PENDING

    @workflow.run
    async def process_order(self, order: OrderRequest) -> OrderResult:
        opts = {"start_to_close_timeout": timedelta(seconds=5)}

        # Payment
        await workflow.execute_activity("ChargePayment", args=[order.order_id, order.amount], **opts)

        # Inventory
        await workflow.execute_activity("ReserveInventory", args=[order.item, order.quantity], **opts)

        # Dispatch
        await workflow.execute_activity("DispatchShipping", args=[order.order_id, order.address], **opts)

        # Notification
        await workflow.execute_activity(
            "SendNotification", args=[order.order_id, "DISPATCHED", "Dispatched!"], **opts
        )

        self._status = OrderStatus.COMPLETED

        return OrderResult.success(order.order_id)

    @workflow.query(name="getStatus")
    def get_status(self) -> OrderStatus:
        return self._status
