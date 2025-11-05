CREATE TABLE IF NOT EXISTS event_logs (
    id VARCHAR(36) PRIMARY KEY,
    trigger_id VARCHAR(36) NULL,  -- NULL for manual/test runs without persisted trigger
    trigger_type ENUM('webhook', 'time_scheduled', 'cron_scheduled') NOT NULL,
    fired_at DATETIME NOT NULL,
    payload JSON NULL,
    source ENUM('webhook', 'scheduler', 'manual-test') NOT NULL,
    execution_status ENUM('success', 'failure') NOT NULL DEFAULT 'success',
    error_message TEXT NULL,  -- Populated on failure
    retention_status ENUM('active', 'archived', 'deleted') NOT NULL DEFAULT 'active',
    is_test_run BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_fired_at (fired_at),
    INDEX idx_trigger_id (trigger_id),
    INDEX idx_retention_status (retention_status),
    INDEX idx_execution_status (execution_status),
    FOREIGN KEY (trigger_id) REFERENCES triggers(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
