# Defer unused event and notification abstractions

Phase 0 ships the working Inbound Event pipeline and only the validation and serialization shape of a future Domain Event. It does not create a publisher, event bus, domain-events table, or notification interface until a real consumer exists.

## Phase 0 amendment

The approved specification adds a transport-neutral, versioned export contract shell with opaque payloads and shared fixtures. It does not add a live export route, persistent service, OCR/accounting behavior, notification abstraction, publisher, event bus, or production Domain Event type. Those remain deferred until a real consumer exists.
