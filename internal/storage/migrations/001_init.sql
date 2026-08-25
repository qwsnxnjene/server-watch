CREATE TABLE IF NOT EXISTS metrics (
                         id INTEGER PRIMARY KEY AUTOINCREMENT,
                         ts TIMESTAMP NOT NULL,
                         cpu REAL NOT NULL,
                         mem_used_mb REAL NOT NULL,
                         mem_total_mb REAL NOT NULL,
                         disk_used_gb REAL NOT NULL,
                         disk_total_gb REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS alerts (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        ts TIMESTAMP NOT NULL,
                        type TEXT CHECK(type IN ('HIGH_CPU','HIGH_MEM')),
                        threshold REAL NOT NULL,
                        value REAL NOT NULL,
                        resolved BOOLEAN NOT NULL DEFAULT FALSE,
                        resolved_ts TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_metrics_ts ON metrics(ts);
CREATE INDEX IF NOT EXISTS idx_alerts_ts ON alerts(ts);
CREATE INDEX IF NOT EXISTS idx_alerts_type_resolved ON alerts(type, resolved);