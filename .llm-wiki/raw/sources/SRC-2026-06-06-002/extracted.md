HMAC request signing is a stateless authentication mechanism where a shared secret key is established once per consumer at creation time. Instead of sending the secret with each request (like an API key), the consumer signs the request payload using HMAC-SHA256 and sends the signature as a header. The server recomputes the signature using the consumer's stored secret and verifies it matches.

Key security properties:
- Secret never travels over the wire (only the signature does)
- Request integrity — body is part of the signature, so payload tampering is detectable
- Replay protection — a timestamp is included in the signed message; server rejects requests older than a configurable window
- Stateless — no session, no token issuance, no rotation lifecycle
- No dependencies — HMAC is built into every standard library

Headers:
- X-Consumer-ID: identifies the consumer
- X-Timestamp: unix timestamp of when the request was signed
- X-Signature: hex-encoded HMAC-SHA256 of "consumer_id:timestamp:canonical_body"

This is the recommended approach for Notifier because it eliminates the need for API key rotation entirely — the secret never changes, and the authentication is per-request and self-validating.