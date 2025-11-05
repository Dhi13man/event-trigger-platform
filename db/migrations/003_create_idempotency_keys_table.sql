-- Create idempotency_keys table
CREATE TABLE IF NOT EXISTS idempotency_keys (
    job_id VARCHAR(36) PRIMARY KEY,
    event_id VARCHAR(36) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
