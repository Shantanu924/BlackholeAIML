# Day 2: Cross-Chain Proof, Latency, Failure Injection

* **Hardest failure mode you found:** 
  The hardest failure mode was managing state consistency when the target chain RPC abruptly disconnected during the middle of a batch relay, leaving partial state commitments.

* **Who/what unblocked you:** 
  Nihal's suggestion to implement a Dead-Letter Queue (DLQ) file completely unblocked the retry loop mechanics and allowed us to gracefully recover from network drops.

* **One explicit risk left in config with mitigation noted:** 
  The failure injection flags (e.g. `--inject-drop`) are exposed in the binary. To mitigate this in prod, they are gated by a compiler flag and default to 0 to prevent accidental disruption.
