# Day 3: Integration Hardening with Token/DEX + gRPC

* **Trade-off you made in schema design:** 
  I chose a generic `meta map` instead of strongly typed fields for DEX/Token specific attributes to allow rapid iteration, sacrificing some type safety for cross-module flexibility.

* **Credit to Token/DEX teammates’ formats:** 
  Yashika and Nihal provided exact JSON samples for their respective modules, which made parsing their payloads trivial and aligned our schemas immediately.

* **One place you resisted over-engineering and documented a follow-up:** 
  I resisted building a full gRPC streaming service for logs, settling for a simple REST polling endpoint with a limit parameter. A follow-up for WebSocket streaming is documented in bridge-api.md.
