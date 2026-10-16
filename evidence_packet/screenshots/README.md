# Visual Verification & UI Demonstrations

This directory references the graphical and visual dashboards demonstrating the live VAJRA system:

1. **Wallet Dashboard Web UI:**
   - Source: `ui/wallet-dashboard/static/index.html` and `ui/wallet-dashboard/static/login.html`
   - Features: Pravesh SSO Login, Real-time BHIV balance, Send/Receive transaction modal, Transaction history table.
2. **Bridge SDK Dashboard UI:**
   - Source: `ui/bridge-dashboard/static/index.html`
   - Features: Cross-chain asset lock & mint telemetry, Polygon Amoy contract status.
3. **Grafana / Prometheus Health:**
   - Scrape route: `http://localhost:8080/api/metrics`
   - Unified Health: `http://localhost:8080/api/monitoring/unified`
