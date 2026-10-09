from fastapi import APIRouter, Request, Response
from fastapi.responses import JSONResponse

from app.infra import DownstreamServiceError, OutOfStockError
from app.models import OrderRequest

router = APIRouter()


@router.post("/orders", status_code=202)
def create_naive_order(order: OrderRequest, request: Request) -> Response:
    """Processes an order with plain synchronous calls to the downstream services:
    no Temporal, no retries, no compensation."""
    services = request.app.state.services
    services.charge_payment(order.order_id, order.amount)
    services.reserve_inventory(order.item, order.quantity)
    services.dispatch_shipping(order.order_id, order.address)
    services.send_notification(order.order_id, "DISPATCHED", "Dispatched!")
    return Response(status_code=202)


def downstream_error_handler(_: Request, ex: DownstreamServiceError) -> JSONResponse:
    if ex.status is None:
        status = 503  # unreachable
    elif ex.status >= 500:
        status = 502  # the service's own failure, not our caller's
    else:
        status = ex.status
    return JSONResponse(status_code=status, content={"error": str(ex)})


def out_of_stock_handler(_: Request, ex: OutOfStockError) -> JSONResponse:
    return JSONResponse(status_code=400, content={"error": str(ex)})
