# Notifier Service

Real-time WebSocket gateway for **`saga.orchestrator.state.changed`** events from Kafka to frontend clients.

## Overview

* **Consumes**: `saga.orchestrator.state.changed` (Kafka)
* **Publishes**: WebSocket messages to clients subscribed by `saga_id`
* **Stateless**: no DB, only in-memory connection registry
* **Configurable** via `config.toml` + env (Viper)

---

## 1. Setup

### Requirements

* Go 1.24+
* Kafka cluster
* Config file (`config.toml`) in project root

### Example `config.toml`

```toml
[http]
addr = ":8888"
allowed_origins = ["http://localhost:3000"]

[kafka]
brokers = ["kafka:9092"]
group_id = "notifier"
topic = "saga.orchestrator.state.changed"

[ws]
ping_interval_sec = 20
write_timeout_sec = 10
max_message_bytes = 1048576

[security]
require_auth = false
```

### Run

```sh
go build -o notifier ./cmd/notifier
./notifier
```

Or with Docker:

```sh
docker build -t notifier .
docker run --network=host -v $(pwd)/config.toml:/app/config.toml notifier
```

---

## 2. WebSocket API

**Endpoint**

```
GET /ws?saga_id=<uuid>
```

Optional headers:

* `Authorization: Bearer <token>` (if auth enabled)
* `X-Tenant-ID: <tenant>` (for tenant filtering)

**Behavior**:

* Opens a WebSocket tied to `saga_id`
* Sends every matching Kafka event to the client in JSON format

---

## 3. Client-Side Example

```js
const sagaId = '3f8e7a4b-...';
const ws = new WebSocket(`ws://localhost:8888/ws?saga_id=${sagaId}`);

ws.onmessage = (e) => {
  const evt = JSON.parse(e.data);
  console.log('Saga update:', evt.payload.status, evt.payload.step);
};

ws.onclose = () => console.log('Connection closed');
```

---

## 4. Scaling Notes

* **Single instance (MVP)**: simplest, all events consumed & routed in one place.
* **Multiple instances**:

    * Use a shared Kafka consumer group + sticky routing by `saga_id`, OR
    * No group (fan-out mode) to let each instance consume all events.
