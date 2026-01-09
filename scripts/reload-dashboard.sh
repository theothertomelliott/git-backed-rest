#!/bin/bash

# Reload Grafana dashboard configuration
# This script reloads dashboards from disk without restarting Grafana

set -e

echo "🔄 Reloading Grafana dashboard configuration..."

# Check if Grafana is running
if ! curl -s http://localhost:3000/api/health > /dev/null; then
    echo "❌ Grafana is not running at http://localhost:3000"
    echo "💡 Start Grafana first with: docker-compose up -d grafana"
    exit 1
fi

# Reload dashboards from provisioning
echo "📊 Reloading dashboards from provisioning..."
response=$(curl -s -w "%{http_code}" -o /dev/null -X POST \
    http://admin:admin@localhost:3000/api/admin/provisioning/dashboards/reload)

if [ "$response" = "200" ]; then
    echo "✅ Dashboard configuration reloaded successfully!"
    echo ""
    echo "📈 Dashboard: Git-Backed-REST API Dashboard"
    echo "🔗 Grafana: http://localhost:3000 (admin/admin)"
    echo ""
    echo "💡 If you don't see changes, try:"
    echo "   1. Refresh your browser (Ctrl+F5 or Cmd+Shift+R)"
    echo "   2. Clear browser cache"
    echo "   3. Check the dashboard list in Grafana"
else
    echo "❌ Failed to reload dashboard configuration (HTTP $response)"
    echo "🔧 Troubleshooting:"
    echo "   - Check if Grafana admin credentials are correct"
    echo "   - Verify Grafana is accessible at http://localhost:3000"
    echo "   - Check Grafana logs: docker-compose logs grafana"
    exit 1
fi

echo ""
echo "🎉 Dashboard reload complete!"
