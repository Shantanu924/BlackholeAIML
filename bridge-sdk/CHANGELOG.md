# Changelog

All notable changes to the BlackHole Bridge SDK will be documented in this file.

## [0.3-rc1] - 2026-06-27

### Added
- **Day 4: Cryptographic Signature Verification**:
  - Implemented full Ed25519 signature checks using raw byte slices for signature payloads.
  - Added public key registration mapping to verify transaction signatures deterministically.
- **Day 4: Replay Attack Protection**:
  - Built thread-safe nonce tracking caching map (`lastSeenNonces`) to protect transactions from cross-channel replay attempts.
  - Returns `400 Bad Request` on unsigned, invalid-signature, or replay-detection scenarios.
- **Day 4: Merkle Proof & Attestation Bundles**:
  - Designed the Attestation module in `core/attest` writing transaction signatures, hash metadata, and target chain status alongside Merkle Event Roots (every 10 events).
- **Day 3: CLI Event Tailing**:
  - Integrated `bridgectl tail` diagnostic command to follow live events with correct parsing of structured JSON data.
- **Day 3: HTTP/gRPC Client Demo**:
  - Provided a unified HTTP REST / gRPC client demonstration script under `examples/grpc/relay_client.go`.
- **Day 3: Token and DEX Routing**:
  - Added module labels (`TOKEN`, `DEX`, `STAKE`) for message routing payloads alongside metadata mappings.

### Changed
- Re-routed Nginx proxy configuration to balance API requests across three instances with active failover parameters.
- Configured automated periodic backup schedules for persistent BoltDB cache files and system logs.

### Fixed
- Fixed Go unit test compilation and validation errors in `tests/signature_test.go` by transitioning inputs to byte slices.
