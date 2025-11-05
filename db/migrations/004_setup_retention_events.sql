-- Enable MySQL Event Scheduler (for automatic retention management)
SET GLOBAL event_scheduler = ON;

-- ============================================
-- Retention Event 1: Archive Active Events
-- ============================================
-- Runs every 5 minutes to move events from 'active' to 'archived' after 2 hours

DELIMITER $$

CREATE EVENT IF NOT EXISTS archive_old_events
ON SCHEDULE EVERY 5 MINUTE
DO
BEGIN
    UPDATE event_logs
    SET retention_status = 'archived'
    WHERE retention_status = 'active'
    AND fired_at < DATE_SUB(NOW(), INTERVAL 2 HOUR);

    -- Log the archival (optional, for monitoring)
    -- INSERT INTO retention_log (action, affected_rows, executed_at)
    -- VALUES ('archive', ROW_COUNT(), NOW());
END$$

DELIMITER ;

-- ============================================
-- Retention Event 2: Delete Old Events
-- ============================================
-- Runs every 10 minutes to permanently delete events older than 48 hours

DELIMITER $$

CREATE EVENT IF NOT EXISTS delete_old_events
ON SCHEDULE EVERY 10 MINUTE
DO
BEGIN
    DELETE FROM event_logs
    WHERE fired_at < DATE_SUB(NOW(), INTERVAL 48 HOUR);

    -- Log the deletion (optional, for monitoring)
    -- INSERT INTO retention_log (action, affected_rows, executed_at)
    -- VALUES ('delete', ROW_COUNT(), NOW());
END$$

DELIMITER ;

-- ============================================
-- Retention Event 3: Clean Old Idempotency Keys
-- ============================================
-- Runs daily to clean up idempotency keys older than 7 days

DELIMITER $$

CREATE EVENT IF NOT EXISTS cleanup_idempotency_keys
ON SCHEDULE EVERY 1 DAY
DO
BEGIN
    DELETE FROM idempotency_keys
    WHERE created_at < DATE_SUB(NOW(), INTERVAL 7 DAY);
END$$

DELIMITER ;

-- ============================================
-- Optional: Retention Log Table
-- ============================================
-- Uncomment to track retention operations

-- CREATE TABLE IF NOT EXISTS retention_log (
--     id INT AUTO_INCREMENT PRIMARY KEY,
--     action VARCHAR(20) NOT NULL,
--     affected_rows INT NOT NULL,
--     executed_at DATETIME NOT NULL,
--     INDEX idx_executed_at (executed_at)
-- ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Verify Events Are Created
-- ============================================
-- Query to check active events:
-- SELECT event_name, event_definition, status, interval_value, interval_field
-- FROM information_schema.events
-- WHERE event_schema = 'event_trigger';
