#!/bin/bash
set -e

echo "📊 Setting up Temporal Grafana Dashboards..."

DASHBOARDS_DIR="monitoring/grafana/dashboards"
TEMP_DIR="/tmp/temporal-dashboards"

# Create dashboards directory
mkdir -p "$DASHBOARDS_DIR"

# Clone the official dashboards repo
echo "Downloading official Temporal dashboards..."
if [ -d "$TEMP_DIR" ]; then
    rm -rf "$TEMP_DIR"
fi

git clone --depth 1 https://github.com/temporalio/dashboards.git "$TEMP_DIR"

# Copy server dashboards
echo "Installing server dashboards..."
cp "$TEMP_DIR"/server/*.json "$DASHBOARDS_DIR/" 2>/dev/null || echo "No server dashboards found"

# Copy SDK dashboards (if you want them)
echo "Installing SDK dashboards..."
mkdir -p "$DASHBOARDS_DIR/sdk"
cp "$TEMP_DIR"/sdk/*.json "$DASHBOARDS_DIR/sdk/" 2>/dev/null || echo "No SDK dashboards found"

# Cleanup
rm -rf "$TEMP_DIR"

echo "✅ Dashboards installed successfully!"
echo ""
echo "Dashboards available:"
ls -1 "$DASHBOARDS_DIR"/*.json 2>/dev/null | xargs -n1 basename || echo "No dashboards found"
echo ""
echo "SDK Dashboards:"
ls -1 "$DASHBOARDS_DIR/sdk"/*.json 2>/dev/null | xargs -n1 basename || echo "No SDK dashboards found"
