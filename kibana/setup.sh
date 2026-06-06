#!/bin/sh
set -eu

KIBANA="${KIBANA_URL:-http://kibana:5601}"

cat > /tmp/data-view.json << 'EOF'
{
  "data_view": {
    "id": "app-logs",
    "title": "app-logs-*",
    "timeFieldName": "@timestamp"
  }
}
EOF

cat > /tmp/saved-search.json << 'EOF'
{
  "attributes": {
    "title": "App Logs",
    "columns": [
      "app.level",
      "app.msg",
      "app.request_id",
      "app.method",
      "app.path",
      "app.status",
      "app.duration_ms"
    ],
    "sort": [["@timestamp", "desc"]],
    "kibanaSavedObjectMeta": {
      "searchSourceJSON": "{\"index\":\"app-logs\",\"query\":{\"query\":\"\",\"language\":\"kuery\"},\"filter\":[]}"
    }
  },
  "references": [{
    "name": "kibanaSavedObjectMeta.searchSourceJSON.index",
    "type": "index-pattern",
    "id": "app-logs"
  }]
}
EOF

echo "Creating data view..."
curl -sf -X POST "$KIBANA/api/data_views/data_view" \
  -H "kbn-xsrf: true" \
  -H "Content-Type: application/json" \
  -d @/tmp/data-view.json || echo "Data view may already exist, continuing."

echo "Creating saved search..."
curl -sf -X POST "$KIBANA/api/saved_objects/search/app-logs-search?overwrite=true" \
  -H "kbn-xsrf: true" \
  -H "Content-Type: application/json" \
  -d @/tmp/saved-search.json

cat > /tmp/dashboard.json << 'EOF'
{
  "attributes": {
    "title": "App Logs",
    "panelsJSON": "[{\"type\":\"search\",\"gridData\":{\"x\":0,\"y\":0,\"w\":48,\"h\":20,\"i\":\"1\"},\"panelIndex\":\"1\",\"embeddableConfig\":{},\"panelRefName\":\"panel_1\"}]",
    "timeRestore": false,
    "kibanaSavedObjectMeta": {
      "searchSourceJSON": "{\"query\":{\"query\":\"\",\"language\":\"kuery\"},\"filter\":[]}"
    }
  },
  "references": [{
    "name": "panel_1",
    "type": "search",
    "id": "app-logs-search"
  }]
}
EOF

echo "Creating dashboard..."
curl -sf -X POST "$KIBANA/api/saved_objects/dashboard/app-logs-dashboard?overwrite=true" \
  -H "kbn-xsrf: true" \
  -H "Content-Type: application/json" \
  -d @/tmp/dashboard.json

echo "Kibana setup complete."
