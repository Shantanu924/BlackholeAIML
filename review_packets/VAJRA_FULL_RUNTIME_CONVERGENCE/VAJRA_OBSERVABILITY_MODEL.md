# VAJRA Observability Model (InsightFlow / Pravah)

To guarantee operational transparency and immediate failure detection, the VAJRA Trust Layer integrates directly with the TANTRA InsightFlow/Pravah observability ecosystem.

## 1. Telemetry Generation
The VAJRA network emits three distinct streams of telemetry:
1. **Metrics:** Real-time block height, transaction throughput (TPS), pending mempool size, active validator count, and node memory usage.
2. **Logs:** Structured JSON application logs tracing critical lifecycle events (startup, peer discovery, consensus, block finality, bridge deposits).
3. **Traces:** Cross-service distributed tracing (e.g., tracking a user's wallet request as it hits the Nginx reverse proxy $\rightarrow$ Node 1 RPC $\rightarrow$ P2P Network Gossip).

## 2. Infrastructure Integration
Because the production topology is hosted on AWS ECS Fargate:
* **Log Aggregation:** We utilize the native `awslogs` driver configured in our `deploy/mainnet/main.tf` task definitions.
* **Ingestion:** AWS CloudWatch serves as the primary log aggregator, which is then forwarded securely to the InsightFlow Grafana/Prometheus endpoints via an automated Lambda firehose.

## 3. Critical Alerts
The Fraud Service actively monitors the network for malicious behavior (e.g., duplicate block proposals). If a threshold is exceeded:
* An HTTP POST is fired to the Pravah Incident webhook.
* The system automatically generates an incident report, alerting the Governance Council of the threat and the corresponding Validator IP address that was slashed.
