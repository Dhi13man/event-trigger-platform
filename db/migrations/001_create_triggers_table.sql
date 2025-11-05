CREATE TABLE IF NOT EXISTS triggers (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type ENUM('webhook', 'time_scheduled', 'cron_scheduled') NOT NULL,
    status ENUM('active', 'inactive') NOT NULL DEFAULT 'active',
    config JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_type (type),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create trigger_schedules table for future executions
CREATE TABLE IF NOT EXISTS trigger_schedules (
    id VARCHAR(36) PRIMARY KEY,
    trigger_id VARCHAR(36) NOT NULL,
    fire_at DATETIME NOT NULL,
    status ENUM('pending', 'processing', 'completed', 'cancelled') NOT NULL DEFAULT 'pending',
    attempt_count INT NOT NULL DEFAULT 0,
    last_attempt_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_fire_at_status (fire_at, status),
    INDEX idx_trigger_id (trigger_id),
    CONSTRAINT fk_trigger_schedules_trigger FOREIGN KEY (trigger_id) REFERENCES triggers(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
