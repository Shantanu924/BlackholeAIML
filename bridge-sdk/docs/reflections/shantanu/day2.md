# Daily Reflection - Day 2
**Author:** Shantanu  
**Date:** June 25, 2026

## Reflection
During Day 2, my focus was on implementing the rolling Merkle hash verification mechanism and establishing detailed performance latency tracing. Writing the logic to automatically compute and flush `EventRoot` hashes every 10 events allowed us to construct reliable cross-chain proof anchors. Integrating failure injection controls (`--inject-drop` and `--inject-delay-ms`) was challenging but vital to prove that our retry queue and exponential backoff loops can successfully recover messages under network drops. Finally, packaging these dependencies into a multi-stage Docker environment verified that the bridge can run as a sandboxed, low-overhead container, establishing a firm base for production-grade load testing.
