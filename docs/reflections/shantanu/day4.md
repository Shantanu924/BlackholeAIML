# Day 4: Signed Messages, Attestations, and Pre-Prod

* **One assumption you corrected after testing:** 
  I assumed Ed25519 signature verification would be extremely fast, but caching the public key was still necessary to prevent CPU starvation under high load.

* **Acknowledge any library or prior art used for signatures:** 
  I heavily referenced the standard `crypto/ed25519` package and Cosmos SDK's signature verification flow to ensure standard compliance.

* **Clear statement of what remains before true prod traffic:** 
  Before true production traffic, the attestation stub must be replaced with a robust threshold signature scheme (TSS) validator network to avoid single points of failure.
