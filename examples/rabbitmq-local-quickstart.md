# RabbitMQ Event Pipeline — Local Docker Compose Quickstart

A simple DAG that consumes events from a RabbitMQ queue, processes them with transforms, and writes the output to both a JSON file (append) and SQLite.

```
  ┌─────────────┐      ┌─────────────────┐      ┌──────────────┐
  │  RabbitMQ    │─────▶│   etlctl        │─────▶│ events.json  │
  │  (orders)    │      │   listen        │      └──────────────┘
  └─────────────┘      │   ─ uppercase    │      ┌──────────────┐
                       │   ─ trim|title   │─────▶│ events.db    │
                       └─────────────────┘      │ (sqlite3)    │
                                                 └──────────────┘
```

---

## 1. Project Layout

Create the following files alongside your existing `etlctl` project:

```
examples/rabbitmq-local/
├── docker-compose.yaml
├── Dockerfile
├── etls/
│   └── rabbitmq-events.yaml
└── scripts/
    └── publish-test-events.sh
```

---

## 2. ETL Definition

**`etls/rabbitmq-events.yaml`**

```yaml
name: rabbitmq-events

sources:
  - name: order_queue
    type: rabbitmq
    connection:
      url: amqp://guest:guest@rabbitmq:5672/
      queue: orders
      prefetch_count: "10"
      timeout: "5s"

targets:
  - name: json_log
    type: json
    connection:
      filepath: /data/events.json
  - name: events_db
    type: sqlite3
    connection:
      filepath: /data/events.db

pipelines:
  - name: process_orders
    sources: [order_queue]
    targets: [json_log, events_db.orders]
    fields:
      - { source: order_id,   target: order_id }
      - { source: customer,   target: customer_name, transform: "trim|title" }
      - { source: item,       target: item,          transform: "uppercase" }
      - { source: quantity,    target: quantity }
      - { source: total,      target: total }
      - { source: timestamp,  target: event_time }

queries: []
```

Key points:
- `url` uses the Docker Compose service name `rabbitmq` as the hostname.
- The queue `orders` is auto-declared by etlctl on connect (durable, not auto-delete).
- Both targets write to `/data/` which is a mounted Docker volume for persistence.

---

## 3. Dockerfile

**`Dockerfile`**

```dockerfile
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o etlctl ./pkg/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/etlctl .
COPY examples/rabbitmq-local/etls/ /etc/etlctl/configs/
ENTRYPOINT ["./etlctl"]
```

Build context is the repo root so it picks up `go.mod`, `go.sum`, and `pkg/`.

---

## 4. Docker Compose

**`docker-compose.yaml`**

```yaml
services:
  rabbitmq:
    image: rabbitmq:3-management-alpine
    ports:
      - "5672:5672"    # AMQP
      - "15672:15672"  # Management UI
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "check_port_connectivity"]
      interval: 5s
      timeout: 10s
      retries: 10

  etlctl:
    build:
      context: ../..          # repo root
      dockerfile: examples/rabbitmq-local/Dockerfile
    command: ["listen", "rabbitmq-events", "--config-dir", "/etc/etlctl/configs"]
    volumes:
      - etl-data:/data
    depends_on:
      rabbitmq:
        condition: service_healthy

volumes:
  etl-data:
```

Notes:
- `listen` mode keeps etlctl running continuously, consuming from the queue in a loop.
- The `etl-data` volume persists `events.json` and `events.db` across container restarts.
- RabbitMQ management UI is available at http://localhost:15672 (guest/guest).

---

## 5. Test Event Publisher

**`scripts/publish-test-events.sh`**

```bash
#!/usr/bin/env bash
# Publishes sample order events to the RabbitMQ queue.
# Requires: curl (uses the RabbitMQ HTTP API on the management port).

RABBIT_URL="http://localhost:15672/api/exchanges/%2f/amq.default/publish"
AUTH="guest:guest"

publish() {
  local payload="$1"
  curl -s -u "$AUTH" -H "content-type: application/json" \
    -d "{
      \"properties\": {},
      \"routing_key\": \"orders\",
      \"payload\": \"$(echo "$payload" | sed 's/"/\\"/g')\",
      \"payload_encoding\": \"string\"
    }" "$RABBIT_URL" > /dev/null
  echo "Published: $payload"
}

publish '{"order_id":"1001","customer":" alice johnson ","item":"margherita pizza","quantity":"2","total":"25.98","timestamp":"2026-03-15T10:00:00Z"}'
publish '{"order_id":"1002","customer":"bob smith","item":"pepperoni pizza","quantity":"1","total":"14.99","timestamp":"2026-03-15T10:01:00Z"}'
publish '{"order_id":"1003","customer":" carol white  ","item":"garlic bread","quantity":"3","total":"11.97","timestamp":"2026-03-15T10:02:00Z"}'
publish '{"order_id":"1004","customer":"dave brown","item":"caesar salad","quantity":"1","total":"9.99","timestamp":"2026-03-15T10:03:00Z"}'
publish '{"order_id":"1005","customer":" eve davis ","item":"tiramisu","quantity":"2","total":"17.98","timestamp":"2026-03-15T10:04:00Z"}'

echo ""
echo "Done. Events should appear in etlctl output within ~5 seconds."
```

```bash
chmod +x scripts/publish-test-events.sh
```

---

## 6. Run It

```bash
cd examples/rabbitmq-local

# Start everything
docker compose up --build -d

# Watch etlctl logs
docker compose logs -f etlctl

# In another terminal, publish test events
./scripts/publish-test-events.sh
```

You should see etlctl log each batch it processes. After the timeout window (5s), it extracts all queued messages, applies transforms, and loads them to both targets.

---

## 7. Verify Output

### Check the JSON file

```bash
docker compose exec etlctl cat /data/events.json
```

Expected output (formatted):
```json
[
  {
    "order_id": "1001",
    "customer_name": "Alice Johnson",
    "item": "MARGHERITA PIZZA",
    "quantity": "2",
    "total": "25.98",
    "event_time": "2026-03-15T10:00:00Z"
  },
  ...
]
```

Note: `customer` has been trimmed and title-cased, `item` is uppercased.

### Check the SQLite database

```bash
docker compose exec etlctl sh -c "
  apk add --no-cache sqlite > /dev/null 2>&1
  sqlite3 /data/events.db 'SELECT * FROM orders;'
"
```

### RabbitMQ Management UI

Open http://localhost:15672 (guest/guest) to see queue depth, consumer count, and message rates.

---

## 8. Tear Down

```bash
docker compose down           # stop containers
docker compose down -v        # stop + remove the data volume
```

---

## How It Works

1. **etlctl listen** enters a continuous loop, calling `Extract()` on the RabbitMQ source each iteration.
2. The RabbitMQ source consumes messages until the `timeout` (5s) elapses with no new messages, then returns the batch.
3. Field transforms are applied: `trim|title` on customer, `uppercase` on item.
4. The batch is loaded to both targets in sequence — JSON file and SQLite `orders` table.
5. Messages are ACK'd after successful JSON parse; the loop repeats.

---

## Customization Ideas

| Change | How |
|--------|-----|
| Add dedup on `order_id` | Add `uniqueFields: [order_id]` and `keepLast: true` to the pipeline |
| Add a CSV target | Add a new target with `type: csv` and include it in `targets:` |
| Filter by amount | Add a function that drops rows below a threshold |
| Scale consumers | Run multiple `etlctl` containers — RabbitMQ distributes messages round-robin |
| Add a webhook source too | Add a second source with `type: webhook` and include it in `sources:` |
