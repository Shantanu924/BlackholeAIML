# Day 1: Schemas, Endpoints, and Idempotency

* **One thing you weren't sure about and how you handled it:** 
  I wasn't entirely sure how to handle duplicate cross-chain events since both chains might emit the same transaction hash under replay conditions. I handled this by leveraging a local BoltDB instance keyed by a unique `eventHash` to ensure idempotency and safely reject duplicates before they enter the processing queue.

* **Credit any reference or teammate input:** 
  Shiv's initial token registry design provided great clarity on how the bridge should interface with the core chain state.

* **One limitation you're leaving explicit in code/docs:** 
  The mock listeners (`eth_mock.go`, `sol_mock.go`) are currently hardcoded to generate events at a fixed rate, which doesn't fully mimic real-world chain jitter and block finality delays.
