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
publish '{"order_id":"1006","customer":" john davis ","item":"chicken pizza","quantity":"3","total":"19.98","timestamp":"2026-03-15T10:04:00Z"}'

echo ""
echo "Done. Events should appear in etlctl output within ~5 seconds."
