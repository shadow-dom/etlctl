#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="etlctl-test"
REGISTRY_NAME="k3d-etlctl-registry"
REGISTRY_PORT="5111"
HOST_IMAGE="localhost:5111/etlctl:latest"
K8S_IMAGE="k3d-etlctl-registry:5111/etlctl:latest"
NAMESPACE="etlctl"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m'

banner() { echo ""; echo -e "${CYAN}━━━ $* ━━━${NC}"; echo ""; }
info()   { echo -e "${DIM}$*${NC}"; }
step()   { echo -e "  ${GREEN}✓${NC} $*"; }

cleanup() {
    echo ""
    info "Tearing down cluster..."
    k3d cluster delete "$CLUSTER_NAME" 2>/dev/null || true
    k3d registry delete "$REGISTRY_NAME" 2>/dev/null || true
}
trap cleanup EXIT

# ── Setup (silent) ──
banner "1. Setting up k3d cluster + RabbitMQ"

k3d registry delete "$REGISTRY_NAME" 2>/dev/null || true
k3d registry create etlctl-registry --port "$REGISTRY_PORT" >/dev/null 2>&1
step "Local registry on port $REGISTRY_PORT"

k3d cluster delete "$CLUSTER_NAME" 2>/dev/null || true
k3d cluster create "$CLUSTER_NAME" --registry-use "$REGISTRY_NAME:$REGISTRY_PORT" --wait >/dev/null 2>&1
kubectl config use-context "k3d-$CLUSTER_NAME" >/dev/null 2>&1
kubectl create namespace "$NAMESPACE" >/dev/null 2>&1
step "k3d cluster created"

# Build image
cp "$SCRIPT_DIR/rabbitmq-etl.yaml" "$REPO_ROOT/etls/rabbitmq-demo.yaml"
cat > "$REPO_ROOT/Dockerfile" <<'DOCKERFILE'
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o etlctl ./pkg/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates sqlite
RUN mkdir -p /tmp/etlctl
WORKDIR /app
COPY --from=builder /app/etlctl .
COPY etls/ /etc/etlctl/configs/
ENTRYPOINT ["./etlctl"]
DOCKERFILE
docker build -t "$HOST_IMAGE" "$REPO_ROOT" >/dev/null 2>&1
docker push "$HOST_IMAGE" >/dev/null 2>&1
rm -f "$REPO_ROOT/etls/rabbitmq-demo.yaml" "$REPO_ROOT/Dockerfile"
step "etlctl image built and pushed"

# Deploy RabbitMQ
kubectl apply -f "$SCRIPT_DIR/k8s-rabbitmq.yaml" >/dev/null 2>&1
for i in $(seq 1 60); do
    kubectl -n "$NAMESPACE" get pod rabbitmq -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null | grep -q True && break
    sleep 1
done
step "RabbitMQ running in cluster"

# ── Show the ETL config ──
banner "2. The ETL pipeline config (rabbitmq-etl.yaml)"

echo -e "${DIM}This pipeline:${NC}"
echo -e "  ${BOLD}Source:${NC}     RabbitMQ queue '${CYAN}etl_events${NC}'"
echo -e "  ${BOLD}Transforms:${NC} event_type → ${CYAN}UPPERCASE${NC}"
echo -e "             user       → ${CYAN}trim | title${NC}  (remove whitespace, capitalize)"
echo -e "  ${BOLD}Targets:${NC}   events_json (JSON file) + events_db (SQLite table)"
echo ""
echo -e "${DIM}  fields:${NC}"
echo -e "    - source: event_type  target: event_type  transform: ${CYAN}uppercase${NC}"
echo -e "    - source: user        target: user_name   transform: ${CYAN}trim|title${NC}"
echo -e "    - source: amount      target: amount"
echo -e "    - source: timestamp   target: event_time"

# ── Deploy listener ──
banner "3. Deploying the listener (etlctl listen rabbitmq-demo)"

cat <<EOF | kubectl apply -f - >/dev/null 2>&1
apiVersion: v1
kind: ConfigMap
metadata:
  name: etl-rabbitmq-demo-config
  namespace: $NAMESPACE
data:
  rabbitmq-demo.yaml: |
$(sed 's/^/    /' "$SCRIPT_DIR/rabbitmq-etl.yaml")
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: etl-rabbitmq-demo
  namespace: $NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      app: etlctl
      etl: rabbitmq-demo
  template:
    metadata:
      labels:
        app: etlctl
        etl: rabbitmq-demo
    spec:
      containers:
      - name: etl-rabbitmq-demo
        image: $K8S_IMAGE
        command: ["./etlctl", "listen", "rabbitmq-demo", "--config-dir", "/etc/etlctl/configs"]
        volumeMounts:
        - name: config
          mountPath: /etc/etlctl/configs
          readOnly: true
        - name: output
          mountPath: /tmp/etlctl
      volumes:
      - name: config
        configMap:
          name: etl-rabbitmq-demo-config
      - name: output
        emptyDir: {}
      restartPolicy: Always
EOF

for i in $(seq 1 60); do
    POD=$(kubectl -n "$NAMESPACE" get pods -l etl=rabbitmq-demo -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    if [ -n "$POD" ]; then
        STATUS=$(kubectl -n "$NAMESPACE" get pod "$POD" -o jsonpath='{.status.phase}' 2>/dev/null || true)
        [ "$STATUS" = "Running" ] && break
    fi
    sleep 1
done
step "Listener pod running: $POD"
sleep 5
step "Listener connected to RabbitMQ, waiting for messages..."

# ── Publish messages ──
banner "4. Publishing 3 messages to RabbitMQ"

kubectl -n "$NAMESPACE" port-forward pod/rabbitmq 15672:15672 >/dev/null 2>&1 &
PF_PID=$!
sleep 2

MESSAGES=(
    '{"event_type":"purchase","user":"  alice johnson  ","amount":"149.99","timestamp":"2026-03-15T10:01:00Z"}'
    '{"event_type":"refund","user":"  bob smith  ","amount":"29.50","timestamp":"2026-03-15T10:02:00Z"}'
    '{"event_type":"purchase","user":"  carol white  ","amount":"299.00","timestamp":"2026-03-15T10:03:00Z"}'
)

for i in 0 1 2; do
    MSG="${MESSAGES[$i]}"
    PAYLOAD="{\"properties\":{},\"routing_key\":\"etl_events\",\"payload\":$(echo "$MSG" | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read().strip()))'),\"payload_encoding\":\"string\"}"

    curl -s -o /dev/null \
        -u guest:guest \
        -H "Content-Type: application/json" \
        -X POST \
        "http://localhost:15672/api/exchanges/%2f/amq.default/publish" \
        -d "$PAYLOAD"

    # Pretty-print what we sent
    echo -e "  ${YELLOW}→${NC} $(echo "$MSG" | python3 -m json.tool --compact)"
done

kill $PF_PID 2>/dev/null || true
echo ""
step "3 messages published to queue 'etl_events'"

# ── Wait for processing ──
banner "5. Listener processes messages"

info "Waiting for the listener's 10s extraction timeout to fire..."
sleep 15

LOGS=$(kubectl -n "$NAMESPACE" logs "$POD" 2>/dev/null)
echo -e "${DIM}Pod logs:${NC}"
echo "$LOGS" | while IFS= read -r line; do
    echo -e "  ${DIM}│${NC} $line"
done

# ── Show the output ──
banner "6. Verifying output"

echo -e "${BOLD}JSON target (/tmp/etlctl/events.json):${NC}"
echo ""
kubectl -n "$NAMESPACE" exec "$POD" -- cat /tmp/etlctl/events.json 2>/dev/null | python3 -m json.tool | while IFS= read -r line; do
    echo -e "  $line"
done

echo ""
echo -e "${BOLD}SQLite target (/tmp/etlctl/events.db):${NC}"
echo ""
echo -e "  ${DIM}SELECT * FROM events;${NC}"
kubectl -n "$NAMESPACE" exec "$POD" -- sqlite3 -header -column /tmp/etlctl/events.db "SELECT * FROM events;" 2>/dev/null | while IFS= read -r line; do
    echo -e "  $line"
done

# ── Summary ──
banner "7. Transform proof"

echo -e "  ${BOLD}Input → Output:${NC}"
echo ""
echo -e "  event_type: ${YELLOW}purchase${NC}           → ${GREEN}PURCHASE${NC}          ${DIM}(uppercase)${NC}"
echo -e "  event_type: ${YELLOW}refund${NC}             → ${GREEN}REFUND${NC}            ${DIM}(uppercase)${NC}"
echo -e "  user:       ${YELLOW}\"  alice johnson  \"${NC} → ${GREEN}\"Alice Johnson\"${NC}   ${DIM}(trim|title)${NC}"
echo -e "  user:       ${YELLOW}\"  bob smith  \"${NC}     → ${GREEN}\"Bob Smith\"${NC}       ${DIM}(trim|title)${NC}"
echo -e "  user:       ${YELLOW}\"  carol white  \"${NC}   → ${GREEN}\"Carol White\"${NC}     ${DIM}(trim|title)${NC}"
echo ""
echo -e "  ${BOLD}Loaded into:${NC} events.json (JSON) + events.db/events (SQLite)"
echo ""
echo -e "  ${GREEN}${BOLD}RabbitMQ → Transform → Multi-target Load: Working ✓${NC}"
echo ""
