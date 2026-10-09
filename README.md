# Order Management App

A Temporal-based order management workshop app. An order goes through payment, inventory,
shipping, and notification steps, with saga-style compensation on failure, a human-in-the-loop
approval step before dispatch, and a cancellation grace period.

## Architecture

- **order-service-java** — Spring Boot app (port `8080`) hosting both the original `NaiveOrderService`
  and the Temporal-based `OrderWorkflow`, plus the worker that executes it.
- **downstream-services** — a Node/Express app simulating the Payment (`8081`), Inventory (`8082`),
  Shipping (`8083`), and Notification (`8084`) services the workflow calls out to.
- **docker/docker-compose.yml** — spins up:
  - **Temporal** dev server (gRPC on `7233`, Web UI on `8233`)
  - **Toxiproxy** (admin API on `8474`) with a proxy per downstream service (`8501`–`8504`)
    seeded with latency toxics, so the workflow's timeouts/retries have something to chew on
  - **Toxihub** UI (`7072`) for viewing/toggling those proxies

The order-service talks to the downstream services *through* Toxiproxy (see `services.*-url` in
`order-service-java/src/main/resources/application.yml`), not directly — so start the downstream
services before (or alongside) `docker-compose up`.

## Prerequisites

- Docker (and Docker Compose)
- Java 17 or newer. The Gradle wrapper uses the JDK from
  `JAVA_HOME` (or `java` on your `PATH`), so no Gradle install or extra JDK download is needed
- Node.js 18+
- [HTTPie](https://httpie.io/) (`http`) for the example requests below — any HTTP client works

## 1. Start the infrastructure (Temporal + Toxiproxy)

```bash
cd docker
docker-compose up
```

This brings up Temporal (Web UI at http://localhost:8233) and Toxiproxy/Toxihub
(dashboard at http://localhost:7072).

## 2. Start the downstream services

```bash
cd downstream-services
npm install
npm start
```

This starts the Payment, Inventory, Shipping, and Notification mock services on ports
`8081`–`8084`, which Toxiproxy proxies as `8501`–`8504`.

## 3. Start the order service

```bash
cd order-service-java
./gradlew bootRun
```

The Spring Boot app starts on `8080`, connects to Temporal at `127.0.0.1:7233`, and registers a
worker on the `order-processing` task queue.

## 4. Send requests with HTTPie

### Naive endpoint (no Temporal, direct synchronous calls)

```bash
http POST :8080/orders orderId=order-1 amount:=99.99 item=WIDGET quantity:=2 address="1 Main St"
```

### Temporal-backed endpoint

Start an order workflow:

```bash
http POST :8080/temporal/orders orderId=order-1 amount:=99.99 item=WIDGET quantity:=2 address="1 Main St"
```

Check its status:

```bash
http GET :8080/temporal/orders/order-1/status
```

Approve dispatch (human-in-the-loop signal — required before shipping proceeds):

```bash
http POST :8080/temporal/orders/order-1/approve approverEmail==manager@example.com
```

Cancel an order (signal honored during the cancellation grace period):

```bash
http POST :8080/temporal/orders/order-1/cancel reason=="Customer changed mind"
```

You can also watch the workflow execution live in the Temporal Web UI at http://localhost:8233.

### Simulating a failure

Set `item` to `OUT_OF_STOCK` to make the Inventory mock reject the reservation and trigger saga
compensation:

```bash
http POST :8080/temporal/orders orderId=order-2 amount:=49.99 item=OUT_OF_STOCK quantity:=1 address="2 Main St"
```
