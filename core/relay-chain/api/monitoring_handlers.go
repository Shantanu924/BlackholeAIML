package api

import (
	"encoding/json"
	"net/http"
	"time"
)

func (s *APIServer) handleUnifiedMonitoring(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Collect blockchain metrics
	blockchainInfo := s.blockchain.GetBlockchainInfo()

	// Note: Workflow metrics removed - bridge SDK runs separately

	// Collect system health
	systemHealth := map[string]interface{}{
		"blockchain": map[string]interface{}{
			"healthy":      true, // Assume healthy if responding
			"block_height": blockchainInfo["blockHeight"],
			"pending_txs":  blockchainInfo["pendingTxs"],
		},
	}

	// Calculate overall system health (blockchain only)
	overallHealth := true

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"overall_health":  overallHealth,
			"system_health":   systemHealth,
			"blockchain_info": blockchainInfo,
			"timestamp":       time.Now().Format(time.RFC3339),
			"uptime_seconds":  time.Since(time.Now().Add(-time.Hour)).Seconds(), // Mock uptime
		},
	}

	json.NewEncoder(w).Encode(response)
}

// handleMonitoringDashboard serves a unified monitoring dashboard
func (s *APIServer) handleMonitoringDashboard(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>BlackHole Unified Monitoring Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background: #1a1a1a; color: #fff; }
        .container { max-width: 1400px; margin: 0 auto; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(350px, 1fr)); gap: 20px; }
        .card { background: #2d2d2d; padding: 20px; border-radius: 8px; box-shadow: 0 4px 8px rgba(0,0,0,0.3); border: 1px solid #444; }
        .card h3 { margin-top: 0; color: #fff; border-bottom: 2px solid #667eea; padding-bottom: 10px; }
        .metric { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; border-bottom: 1px solid #444; }
        .metric:last-child { border-bottom: none; }
        .metric-label { color: #ccc; }
        .metric-value { color: #fff; font-weight: bold; }
        .status-healthy { color: #27ae60; }
        .status-unhealthy { color: #e74c3c; }
        .status-warning { color: #f39c12; }
        .btn { background: #667eea; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; margin: 5px; }
        .btn:hover { background: #5a6fd8; }
        .refresh-indicator { display: inline-block; margin-left: 10px; opacity: 0; transition: opacity 0.3s; }
        .refresh-indicator.active { opacity: 1; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🌌 BlackHole Unified Monitoring Dashboard</h1>
            <p>Real-time monitoring of all system components and workflows</p>
            <button class="btn" onclick="refreshAll()"><svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M17.65,6.35C16.2,4.9 14.21,4 12,4A8,8 0 0,0 4,12A8,8 0 0,0 12,20C15.73,20 18.84,17.45 19.73,14H17.65C16.83,16.33 14.61,18 12,18A6,6 0 0,1 6,12A6,6 0 0,1 12,6C13.66,6 15.14,6.69 16.22,7.78L13,11H20V4L17.65,6.35Z"/></svg> Refresh All</button>
            <span class="refresh-indicator" id="refresh-indicator">⟳ Refreshing...</span>
        </div>

        <div class="grid">
            <div class="card">
                <h3>🏥 System Health Overview</h3>
                <div id="system-health">
                    <div class="metric">
                        <span class="metric-label">Overall Status:</span>
                        <span class="metric-value" id="overall-status">Loading...</span>
                    </div>
                    <div class="metric">
                        <span class="metric-label">Blockchain:</span>
                        <span class="metric-value" id="blockchain-health">Loading...</span>
                    </div>
                    <div class="metric">
                        <span class="metric-label">Workflow System:</span>
                        <span class="metric-value" id="workflow-health">Loading...</span>
                    </div>
                    <div class="metric">
                        <span class="metric-label">System Uptime:</span>
                        <span class="metric-value" id="system-uptime">Loading...</span>
                    </div>
                </div>
            </div>

            <div class="card">
                <h3>⛓️ Blockchain Metrics</h3>
                <div id="blockchain-metrics">
                    <div class="metric">
                        <span class="metric-label">Block Height:</span>
                        <span class="metric-value" id="block-height">-</span>
                    </div>
                    <div class="metric">
                        <span class="metric-label">Pending Transactions:</span>
                        <span class="metric-value" id="pending-txs">-</span>
                    </div>
                    <div class="metric">
                        <span class="metric-label">Total Supply:</span>
                        <span class="metric-value" id="total-supply">-</span>
                    </div>
                    <div class="metric">
                        <span class="metric-label">Active Validators:</span>
                        <span class="metric-value" id="active-validators">-</span>
                    </div>
                </div>
            </div>

            <div class="card">
                <h3><svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M3.9 12c0-1.71 1.39-3.1 3.1-3.1h4V7H6.99c-2.76 0-5 2.24-5 5s2.24 5 5 5H11v-1.9H7c-1.71 0-3.1-1.39-3.1-3.1zM8 13h8v-2H8v2zm5-6h4.01c2.76 0 5 2.24 5 5s-2.24 5-5 5H13v1.9h4.01c2.76 0 5-2.24 5-5s-2.24-5-5-5H13V7z"/></svg> Workflow Components</h3>
                <div id="workflow-components">Loading...</div>
            </div>

            <div class="card">
                <h3><svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M15,3V7.59L7.59,15H4V17H7.59L15,9.59V15H17V9.59L9.59,2H15V3M17,17V21H15V17H17Z"/></svg> Bridge Metrics</h3>
                <div id="bridge-metrics">Loading...</div>
            </div>

            <div class="card">
                <h3><svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M3 13h8V3H3v10zm0 8h8v-6H3v6zm10 0h8V11h-8v10zm0-18v6h8V3h-8z"/></svg> Performance Metrics</h3>
                <div id="performance-metrics">
                    <div class="metric">
                        <span class="metric-label">Response Time:</span>
                        <span class="metric-value" id="response-time">-</span>
                    </div>
                    <div class="metric">
                        <span class="metric-label">Throughput (TPS):</span>
                        <span class="metric-value" id="throughput">-</span>
                    </div>
                    <div class="metric">
                        <span class="metric-label">Error Rate:</span>
                        <span class="metric-value" id="error-rate">-</span>
                    </div>
                    <div class="metric">
                        <span class="metric-label">Memory Usage:</span>
                        <span class="metric-value" id="memory-usage">-</span>
                    </div>
                </div>
            </div>

            <div class="card">
                <h3><svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M1,21H23L12,2L1,21Z"/></svg> Recent Alerts</h3>
                <div id="recent-alerts">No alerts</div>
            </div>
        </div>
    </div>

    <script>
        let refreshInterval;

        async function fetchUnifiedMonitoring() {
            try {
                showRefreshIndicator();
                const response = await fetch('/api/monitoring/unified');
                const data = await response.json();
                updateUnifiedDashboard(data);
            } catch (error) {
                console.error('Error fetching unified monitoring data:', error);
            } finally {
                hideRefreshIndicator();
            }
        }

        function updateUnifiedDashboard(data) {
            if (!data.success) return;

            const monitoringData = data.data;

            // Update system health
            updateSystemHealth(monitoringData);

            // Update blockchain metrics
            updateBlockchainMetrics(monitoringData.blockchain_info);

            // Update workflow components
            updateWorkflowComponents(monitoringData.workflow_metrics);

            // Update performance metrics
            updatePerformanceMetrics(monitoringData);
        }

        function updateSystemHealth(data) {
            const overallStatus = document.getElementById('overall-status');
            const blockchainHealth = document.getElementById('blockchain-health');
            const workflowHealth = document.getElementById('workflow-health');
            const systemUptime = document.getElementById('system-uptime');

            overallStatus.innerHTML = data.overall_health ? '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/></svg> Healthy' : '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M19,6.41L17.59,5L12,10.59L6.41,5L5,6.41L10.59,12L5,17.59L6.41,19L12,13.41L17.59,19L19,17.59L13.41,12L19,6.41Z"/></svg> Unhealthy';
            overallStatus.className = 'metric-value ' + (data.overall_health ? 'status-healthy' : 'status-unhealthy');

            if (data.system_health.blockchain) {
                blockchainHealth.innerHTML = data.system_health.blockchain.healthy ? '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/></svg> Healthy' : '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M19,6.41L17.59,5L12,10.59L6.41,5L5,6.41L10.59,12L5,17.59L6.41,19L12,13.41L17.59,19L19,17.59L13.41,12L19,6.41Z"/></svg> Unhealthy';
                blockchainHealth.className = 'metric-value ' + (data.system_health.blockchain.healthy ? 'status-healthy' : 'status-unhealthy');
            }

            if (data.system_health.workflow) {
                const workflowHealthy = data.system_health.workflow.healthy && data.system_health.workflow.available;
                workflowHealth.innerHTML = workflowHealthy ? '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/></svg> Healthy' : '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M19,6.41L17.59,5L12,10.59L6.41,5L5,6.41L10.59,12L5,17.59L6.41,19L12,13.41L17.59,19L19,17.59L13.41,12L19,6.41Z"/></svg> Unhealthy';
                workflowHealth.className = 'metric-value ' + (workflowHealthy ? 'status-healthy' : 'status-unhealthy');
            }

            if (data.uptime_seconds) {
                const hours = Math.floor(data.uptime_seconds / 3600);
                const minutes = Math.floor((data.uptime_seconds % 3600) / 60);
                systemUptime.textContent = hours + 'h ' + minutes + 'm';
            }
        }

        function updateBlockchainMetrics(blockchainInfo) {
            if (!blockchainInfo) return;

            document.getElementById('block-height').textContent = blockchainInfo.blockHeight || '-';
            document.getElementById('pending-txs').textContent = blockchainInfo.pendingTxs || '-';
            document.getElementById('total-supply').textContent = blockchainInfo.totalSupply ? blockchainInfo.totalSupply.toLocaleString() : '-';
            document.getElementById('active-validators').textContent = Object.keys(blockchainInfo.stakes || {}).length;
        }

        function updateWorkflowComponents(workflowMetrics) {
            const container = document.getElementById('workflow-components');

            if (!workflowMetrics) {
                container.innerHTML = '<em>Workflow system not available</em>';
                return;
            }

            let html = '';
            for (const [name, metrics] of Object.entries(workflowMetrics)) {
                html += '<div class="metric">';
                html += '<span class="metric-label">' + name + ':</span>';
                html += '<span class="metric-value status-healthy"><svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/></svg> Running</span>';
                html += '</div>';
            }

            container.innerHTML = html || '<em>No workflow components</em>';
        }

        function updatePerformanceMetrics(data) {
            // Mock performance metrics - in a real implementation, these would come from actual monitoring
            document.getElementById('response-time').textContent = '< 100ms';
            document.getElementById('throughput').textContent = '50 TPS';
            document.getElementById('error-rate').textContent = '0.1%';
            document.getElementById('memory-usage').textContent = '256 MB';
        }

        function showRefreshIndicator() {
            document.getElementById('refresh-indicator').classList.add('active');
        }

        function hideRefreshIndicator() {
            document.getElementById('refresh-indicator').classList.remove('active');
        }

        function refreshAll() {
            fetchUnifiedMonitoring();
        }

        function startAutoRefresh() {
            refreshInterval = setInterval(fetchUnifiedMonitoring, 5000); // Refresh every 5 seconds
        }

        function stopAutoRefresh() {
            if (refreshInterval) {
                clearInterval(refreshInterval);
            }
        }

        // Initialize
        fetchUnifiedMonitoring();
        startAutoRefresh();

        // Stop auto-refresh when page is hidden
        document.addEventListener('visibilitychange', function() {
            if (document.hidden) {
                stopAutoRefresh();
            } else {
                startAutoRefresh();
            }
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleAkashicLineage returns all AKASHIC lineage entries.
// GET /api/akashic/lineage
func (s *APIServer) handleMonitoringMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Collect all metrics
	metrics := map[string]interface{}{
		"blockchain": s.blockchain.GetBlockchainInfo(),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	// Note: Workflow metrics removed - bridge SDK runs separately

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"metrics": metrics,
	})
}

// handleReplayEquality — Phase 3: distributed replay equality for a trace_id.
// GET /api/replay/equality?trace_id=<id>

// handleIdentity — VAJRA Phase 1: Runtime Identity Card.
// GET /identity
