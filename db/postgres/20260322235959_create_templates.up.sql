CREATE TABLE templates
(
    id         BIGSERIAL PRIMARY KEY,
    namespace  TEXT        NOT NULL,
    name       TEXT        NOT NULL,
    content    BYTEA       NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (namespace, name)
);

CREATE INDEX idx_templates_namespace ON templates (namespace);
