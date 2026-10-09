import os
from dataclasses import dataclass


def _env(key: str, default: str) -> str:
    return os.environ.get(key) or default


@dataclass(frozen=True)
class Settings:
    temporal_address: str = _env("TEMPORAL_ADDRESS", "127.0.0.1:7233")
    temporal_namespace: str = _env("TEMPORAL_NAMESPACE", "default")

    # The downstream services are reached through Toxiproxy (8501-8504), not directly.
    payment_url: str = _env("SERVICES_PAYMENT_URL", "http://localhost:8501")
    inventory_url: str = _env("SERVICES_INVENTORY_URL", "http://localhost:8502")
    shipping_url: str = _env("SERVICES_SHIPPING_URL", "http://localhost:8503")
    notification_url: str = _env("SERVICES_NOTIFICATION_URL", "http://localhost:8504")


settings = Settings()
