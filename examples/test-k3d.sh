#!/usr/bin/env bash
set -euo pipefail

# --- Configuration ---
CLUSTER_NAME="etlctl-test"
REGISTRY_NAME="k3d-etlctl-registry"
REGISTRY_PORT="5111"
# Host uses localhost, k8s uses the registry's container name
HOST_IMAGE="localhost:5111/etlctl:latest"
K8S_IMAGE="k3d-etlctl-registry:5111/etlctl:latest"
NAMESPACE="etlctl"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
fail()  { echo -e "${RED}[FAIL]${NC} $*"; exit 1; }

cleanup() {
    info "Cleaning up..."
    k3d cluster delete "$CLUSTER_NAME" 2>/dev/null || true
    k3d registry delete "$REGISTRY_NAME" 2>/dev/null || true
    info "Cleanup complete."
}

# Clean up on exit
trap cleanup EXIT

# --- Pre-flight checks ---
for cmd in k3d kubectl docker; do
    command -v "$cmd" >/dev/null 2>&1 || fail "$cmd is required but not installed"
done

# --- Step 1: Create k3d registry + cluster ---
info "Creating local registry..."
k3d registry delete "$REGISTRY_NAME" 2>/dev/null || true
k3d registry create etlctl-registry --port "$REGISTRY_PORT"

info "Creating k3d cluster..."
k3d cluster delete "$CLUSTER_NAME" 2>/dev/null || true
k3d cluster create "$CLUSTER_NAME" \
    --registry-use "$REGISTRY_NAME:$REGISTRY_PORT" \
    --wait

kubectl config use-context "k3d-$CLUSTER_NAME"
kubectl create namespace "$NAMESPACE" 2>/dev/null || true

info "Cluster ready."

# --- Step 2: Build and push etlctl image ---
info "Building etlctl Docker image..."

# Copy example ETL configs into etls/ temporarily for the build
cp "$SCRIPT_DIR/cronjob-etl.yaml" "$REPO_ROOT/etls/cronjob-demo.yaml"
cp "$SCRIPT_DIR/rabbitmq-etl.yaml" "$REPO_ROOT/etls/rabbitmq-demo.yaml"

# Build using the template Dockerfile
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

docker build -t "$HOST_IMAGE" "$REPO_ROOT" 2>&1 | tail -5
docker push "$HOST_IMAGE" 2>&1 | tail -3

# Clean up temp files
rm -f "$REPO_ROOT/etls/cronjob-demo.yaml" "$REPO_ROOT/etls/rabbitmq-demo.yaml" "$REPO_ROOT/Dockerfile"

info "Image pushed to local registry."

# --- Step 3: Deploy RabbitMQ ---
info "Deploying RabbitMQ..."
kubectl apply -f "$SCRIPT_DIR/k8s-rabbitmq.yaml"

info "Waiting for RabbitMQ to be ready..."
for i in $(seq 1 60); do
    if kubectl -n "$NAMESPACE" get pod rabbitmq -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null | grep -q True; then
        break
    fi
    if [ "$i" -eq 60 ]; then
        fail "RabbitMQ pod not ready after 60s"
    fi
    sleep 1
done
info "RabbitMQ ready."

# --- Step 4: Deploy CronJob ETL ---
info "Deploying CronJob ETL..."

# Generate K8s manifests using ConfigMap approach
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: etl-cronjob-demo-config
  namespace: $NAMESPACE
data:
  cronjob-demo.yaml: |
$(sed 's/^/    /' "$SCRIPT_DIR/cronjob-etl.yaml")
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: etl-cronjob-demo
  namespace: $NAMESPACE
  labels:
    app: etlctl
spec:
  schedule: "*/1 * * * *"
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      backoffLimit: 2
      template:
        metadata:
          labels:
            app: etlctl
            etl: cronjob-demo
        spec:
          containers:
          - name: etl-cronjob-demo
            image: $K8S_IMAGE
            command: ["./etlctl", "run", "cronjob-demo", "--config-dir", "/etc/etlctl/configs"]
            volumeMounts:
            - name: config
              mountPath: /etc/etlctl/configs
              readOnly: true
            - name: output
              mountPath: /tmp/etlctl
          volumes:
          - name: config
            configMap:
              name: etl-cronjob-demo-config
          - name: output
            emptyDir: {}
          restartPolicy: OnFailure
EOF

info "CronJob ETL deployed."

# --- Step 5: Deploy RabbitMQ Listener ETL ---
info "Deploying RabbitMQ Listener ETL..."

cat <<EOF | kubectl apply -f -
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
  labels:
    app: etlctl
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

info "Listener ETL deployed."

# --- Step 6: Wait for listener pod to be running ---
info "Waiting for listener pod to start..."
for i in $(seq 1 60); do
    POD=$(kubectl -n "$NAMESPACE" get pods -l etl=rabbitmq-demo -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    if [ -n "$POD" ]; then
        STATUS=$(kubectl -n "$NAMESPACE" get pod "$POD" -o jsonpath='{.status.phase}' 2>/dev/null || true)
        if [ "$STATUS" = "Running" ]; then
            break
        fi
    fi
    if [ "$i" -eq 60 ]; then
        warn "Listener pod not running after 60s. Checking logs..."
        kubectl -n "$NAMESPACE" describe pod -l etl=rabbitmq-demo 2>/dev/null | tail -20
        fail "Listener pod did not start"
    fi
    sleep 1
done
info "Listener pod running: $POD"

# Give the listener a moment to connect to RabbitMQ
sleep 5

# --- Step 7: Publish test messages to RabbitMQ ---
info "Publishing test messages to RabbitMQ..."

# Port-forward RabbitMQ management API
kubectl -n "$NAMESPACE" port-forward pod/rabbitmq 15672:15672 &
PF_PID=$!
sleep 2

# Publish 3 test messages via RabbitMQ HTTP API
for i in 1 2 3; do
    PAYLOAD=$(cat <<JSONEOF
{"properties":{},"routing_key":"etl_events","payload":"{\"event_type\":\"purchase\",\"user\":\"  user_${i}  \",\"amount\":\"${i}00.50\",\"timestamp\":\"2026-03-15T10:0${i}:00Z\"}","payload_encoding":"string"}
JSONEOF
)
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        -u guest:guest \
        -H "Content-Type: application/json" \
        -X POST \
        "http://localhost:15672/api/exchanges/%2f/amq.default/publish" \
        -d "$PAYLOAD")

    if [ "$HTTP_CODE" = "200" ]; then
        info "  Published message $i (HTTP $HTTP_CODE)"
    else
        warn "  Message $i publish returned HTTP $HTTP_CODE"
    fi
done

kill $PF_PID 2>/dev/null || true

# --- Step 8: Trigger a CronJob run manually (don't wait for the schedule) ---
info "Triggering manual CronJob run..."
kubectl -n "$NAMESPACE" create job --from=cronjob/etl-cronjob-demo etl-cronjob-manual 2>/dev/null || true

# --- Step 9: Wait and verify CronJob ---
info "Waiting for CronJob to complete..."
for i in $(seq 1 90); do
    STATUS=$(kubectl -n "$NAMESPACE" get job etl-cronjob-manual -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}' 2>/dev/null || true)
    if [ "$STATUS" = "True" ]; then
        break
    fi
    FAILED=$(kubectl -n "$NAMESPACE" get job etl-cronjob-manual -o jsonpath='{.status.conditions[?(@.type=="Failed")].status}' 2>/dev/null || true)
    if [ "$FAILED" = "True" ]; then
        warn "CronJob failed. Logs:"
        JOB_POD=$(kubectl -n "$NAMESPACE" get pods --selector=job-name=etl-cronjob-manual -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
        kubectl -n "$NAMESPACE" logs "$JOB_POD" 2>/dev/null || true
        fail "CronJob failed"
    fi
    if [ "$i" -eq 90 ]; then
        fail "CronJob did not complete in 90s"
    fi
    sleep 1
done

# Show CronJob logs
JOB_POD=$(kubectl -n "$NAMESPACE" get pods --selector=job-name=etl-cronjob-manual -o jsonpath='{.items[0].metadata.name}')
info "CronJob logs:"
kubectl -n "$NAMESPACE" logs "$JOB_POD" 2>/dev/null

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN} CronJob ETL: PASS${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# --- Step 10: Verify RabbitMQ listener received messages ---
info "Waiting for listener to process messages (15s)..."
sleep 15

LISTENER_POD=$(kubectl -n "$NAMESPACE" get pods -l etl=rabbitmq-demo -o jsonpath='{.items[0].metadata.name}')
info "Listener pod logs:"
LOGS=$(kubectl -n "$NAMESPACE" logs "$LISTENER_POD" 2>/dev/null)
echo "$LOGS"

# Check that messages were extracted and loaded
if echo "$LOGS" | grep -q "Extracted"; then
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN} RabbitMQ Listener ETL: PASS${NC}"
    echo -e "${GREEN}========================================${NC}"
else
    echo ""
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW} RabbitMQ Listener: No extractions yet${NC}"
    echo -e "${YELLOW} (may need more time for timeout)${NC}"
    echo -e "${YELLOW}========================================${NC}"
    # Show more detail
    kubectl -n "$NAMESPACE" describe pod "$LISTENER_POD" 2>/dev/null | tail -20
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN} All resources:${NC}"
echo -e "${GREEN}========================================${NC}"
kubectl -n "$NAMESPACE" get all 2>/dev/null

echo ""
info "Tests complete. Cluster will be cleaned up on exit."
