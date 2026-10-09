import asyncio
import logging
from concurrent.futures import ThreadPoolExecutor
from contextlib import asynccontextmanager

from fastapi import FastAPI
from temporalio.client import Client
from temporalio.contrib.pydantic import pydantic_data_converter
from temporalio.worker import Worker

from app.api import router
from app.naive import downstream_error_handler, out_of_stock_handler, router as naive_router
from app.config import settings
from app.infra import DownstreamServiceError, DownstreamServices, OutOfStockError
from app.temporal import task_queues
from app.temporal.activities import OrderActivities
from app.temporal.workflows import OrderWorkflow

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s - %(message)s")


@asynccontextmanager
async def lifespan(app: FastAPI):
    client = await Client.connect(
        settings.temporal_address,
        namespace=settings.temporal_namespace,
        data_converter=pydantic_data_converter,
    )
    app.state.temporal_client = client

    services = DownstreamServices(settings)
    app.state.services = services
    activities = OrderActivities(services)
    with ThreadPoolExecutor(max_workers=100) as activity_executor:
        worker = Worker(
            client,
            task_queue=task_queues.ORDER_SERVICE,
            workflows=[OrderWorkflow],
            activities=[
                activities.charge_payment,
                activities.reserve_inventory,
                activities.dispatch_shipping,
                activities.send_notification,
            ],
            activity_executor=activity_executor,
        )
        worker_task = asyncio.create_task(worker.run())
        try:
            yield
        finally:
            await worker.shutdown()
            await worker_task


app = FastAPI(title="order-service-python", lifespan=lifespan)
app.include_router(naive_router)
app.include_router(router)
app.add_exception_handler(DownstreamServiceError, downstream_error_handler)
app.add_exception_handler(OutOfStockError, out_of_stock_handler)
