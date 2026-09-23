CREATE TABLE metrics (
    name VARCHAR(255) PRIMARY KEY,
    mtype TEXT NOT NULL CHECK (mtype IN ('gauge', 'counter')),
    delta BIGINT,
    value DOUBLE PRECISION,
    CHECK (
        (mtype = 'gauge' AND value IS NOT NULL AND delta IS NULL)
        OR
        (mtype = 'counter' AND delta IS NOT NULL AND value IS NULL)
    )
);
