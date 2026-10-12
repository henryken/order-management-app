from fastapi import APIRouter, Query, Request
from temporalio.client import Client

from app.models import OrderRequest, OrderStatus
from app.temporal import task_queues
from app.temporal.workflows import OrderWorkflow

router = APIRouter(prefix="/temporal/orders")


def _workflow_id(order_id: str) -> str:
    return f"order-{order_id}"


def _client(request: Request) -> Client:
    return request.app.state.temporal_client


@router.post("", status_code=202)
async def create_order(order: OrderRequest, request: Request) -> dict[str, str]:
    await _client(request).start_workflow(
        OrderWorkflow.process_order,
        order,
        id=_workflow_id(order.order_id),
        task_queue=task_queues.ORDER_SERVICE,
    )
    return {"workflowId": _workflow_id(order.order_id), "status": "STARTED"}


@router.get("/{order_id}/status")
async def get_order_status(order_id: str, request: Request) -> dict[str, str]:
    handle = _client(request).get_workflow_handle(_workflow_id(order_id))
    status = await handle.query(OrderWorkflow.get_status, result_type=OrderStatus)
    return {"orderId": order_id, "status": status.value}


@router.post("/{order_id}/approve")
async def approve_order(
    order_id: str,
    request: Request,
    approver_email: str = Query("manager@example.com", alias="approverEmail"),
) -> dict[str, str]:
    handle = _client(request).get_workflow_handle(_workflow_id(order_id))
    await handle.signal(OrderWorkflow.approve_dispatch, approver_email)
    return {"message": "Approval signal sent"}


@router.post("/{order_id}/cancel")
async def cancel_order(
    order_id: str,
    request: Request,
    reason: str = "Customer changed mind",
) -> dict[str, str]:
    handle = _client(request).get_workflow_handle(_workflow_id(order_id))
    await handle.signal(OrderWorkflow.cancel_order, reason)
    return {"message": "Cancellation signal sent"}
