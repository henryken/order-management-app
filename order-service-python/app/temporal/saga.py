from typing import Awaitable, Callable


class Saga:
    """Collects compensations and runs them in reverse order. Like the Java Saga helper with its
    default options, compensation is sequential and stops at the first compensation that fails."""

    def __init__(self) -> None:
        self._compensations: list[Callable[[], Awaitable[None]]] = []

    def add_compensation(self, compensation: Callable[[], Awaitable[None]]) -> None:
        self._compensations.append(compensation)

    async def compensate(self) -> None:
        for compensation in reversed(self._compensations):
            await compensation()
