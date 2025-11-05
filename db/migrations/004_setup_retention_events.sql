-- Enable MySQL Event Scheduler for automatic retention management
SET GLOBAL event_scheduler = ON;

-- Archive active events after 2 hours (runs every 5 minutes)
DELIMITER $$

CREATE EVENT IF NOT EXISTS archive_old_events
ON SCHEDULE EVERY 5 MINUTE
DO
BEGIN
    UPDATE event_logs
    SET retention_status = 'archived'
    WHERE retention_status = 'active'
    AND fired_at < DATE_SUB(NOW(), INTERVAL 2 HOUR);
END$$

DELIMITER ;

-- Delete events older than 48 hours (runs every 10 minutes)
DELIMITER $$

CREATE EVENT IF NOT EXISTS delete_old_events
ON SCHEDULE EVERY 10 MINUTE
DO
BEGIN
    DELETE FROM event_logs
    WHERE fired_at < DATE_SUB(NOW(), INTERVAL 48 HOUR);
END$$

DELIMITER ;

-- Clean up idempotency keys older than 7 days (runs daily)
DELIMITER $$

CREATE EVENT IF NOT EXISTS cleanup_idempotency_keys
ON SCHEDULE EVERY 1 DAY
DO
BEGIN
    DELETE FROM idempotency_keys
    WHERE created_at < DATE_SUB(NOW(), INTERVAL 7 DAY);
END$$

DELIMITER ;

-- Optional: Retention log table for tracking operations
-- CREATE TABLE IF NOT EXISTS retention_log (
--     id INT AUTO_INCREMENT PRIMARY KEY,
--     action VARCHAR(20) NOT NULL,
--     affected_rows INT NOT NULL,
--     executed_at DATETIME NOT NULL,
--     INDEX idx_executed_at (executed_at)
-- ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Verify events are created:
-- SELECT event_name, event_definition, status, interval_value, interval_field
-- FROM information_schema.events
-- WHERE event_schema = 'event_trigger';
