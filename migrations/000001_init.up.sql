CREATE TABLE document_cache (
    vin              TEXT PRIMARY KEY,
    payload          JSONB NOT NULL,
    source_signature TEXT NOT NULL,
    cached_at        TIMESTAMPTZ NOT NULL,
    expires_at       TIMESTAMPTZ NOT NULL,
    CONSTRAINT document_cache_ttl_ok CHECK (expires_at >= cached_at)
);

CREATE INDEX document_cache_expires_at_idx ON document_cache (expires_at);

CREATE TABLE search_audit (
    id             BIGSERIAL PRIMARY KEY,
    vin_hash       TEXT NOT NULL,
    vin_suffix     TEXT NOT NULL,
    actor_id       TEXT NOT NULL DEFAULT '',
    request_id     TEXT NOT NULL DEFAULT '',
    trace_id       TEXT NOT NULL DEFAULT '',
    outcome        TEXT NOT NULL CHECK (outcome IN ('OK', 'PARTIAL', 'STALE', 'INVALID_VIN', 'UNAVAILABLE')),
    sources_ok     SMALLINT NOT NULL,
    sources_failed SMALLINT NOT NULL,
    latency_ms     INTEGER NOT NULL,
    requested_at   TIMESTAMPTZ NOT NULL
);

CREATE INDEX search_audit_requested_at_idx ON search_audit (requested_at);
CREATE INDEX search_audit_vin_hash_idx ON search_audit (vin_hash);
