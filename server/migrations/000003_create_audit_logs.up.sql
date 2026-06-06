CREATE TABLE audit_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consumer_id  UUID NOT NULL,
    ip           VARCHAR(45),
    endpoint     VARCHAR(255),
    method       VARCHAR(10),
    status_code  SMALLINT,
    job_id       VARCHAR(255),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_consumer_id ON audit_logs(consumer_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
