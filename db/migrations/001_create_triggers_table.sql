-- Create triggers table
CREATE TABLE IF NOT EXISTS triggers (
    id VARCHAR(36) PRIMARY KEY,
    type ENUM('scheduled', 'api') NOT NULL,
    status ENUM('active', 'inactive') NOT NULL DEFAULT 'active',
    config JSON NOT NULL,
    next_fire_time DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_next_fire_time (next_fire_time),
    INDEX idx_status (status),
    INDEX idx_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
