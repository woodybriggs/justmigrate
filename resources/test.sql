/*
 * ======================================================================================
 * SQLITE ADVANCED SCHEMA: SMART FACTORY IOT SYSTEM
 * ======================================================================================
 * Features demonstrated:
 * - Pragmas (Foreign Keys, Journaling)
 * - Tables: Standard, STRICT (Type enforcement), WITHOUT ROWID (Optimization)
 * - Virtual Tables: FTS5 (Full Text Search)
 * - Columns: Generated (Virtual & Stored), JSON Constraints, Check Constraints
 * - Indexes: Partial Indexes, Indexes on Expressions
 * - Views & Triggers: INSTEAD OF triggers for updatable views
 * ======================================================================================
 */

-- 1. Configuration
-- --------------------------------------------------------------------------------------
PRAGMA foreign_keys = TRUE;            -- Enforce FK constraints
PRAGMA encoding = "UTF-8";           -- Ensure standard encoding

-- Start a transaction to ensure atomic schema creation
BEGIN TRANSACTION;

-- 2. Standard Lookup Table
-- --------------------------------------------------------------------------------------
-- Demonstrates: Standard auto-increment, unique constraints, collation.
CREATE TABLE IF NOT EXISTS device_categories (
    category_id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE, -- 'pump' == 'PUMP'
    description TEXT DEFAULT 'No description provided',
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

-- 3. STRICT Table (SQLite 3.37+)
-- --------------------------------------------------------------------------------------
-- Demonstrates: STRICT mode (rigid typing), JSON validation, Generated Columns.
CREATE TABLE IF NOT EXISTS devices (
    device_id INTEGER PRIMARY KEY,
    category_id INTEGER NOT NULL,
    serial_number TEXT NOT NULL CHECK(length(serial_number) >= 8),
    
    -- JSON Support: SQLite stores JSON as TEXT, but we can enforce validity
    settings_json TEXT CHECK(json_valid(settings_json)),
    
    -- State management
    is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    
    -- Dates stored as TEXT in STRICT mode (ISO8601)
    installed_at TEXT NOT NULL,
    last_service_date TEXT,
    
    -- GENERATED COLUMN (VIRTUAL): Calculated on read, takes no storage space.
    -- Calculates next service date (6 months from last service)
    next_service_due TEXT GENERATED ALWAYS AS (date(last_service_date, '+6 months')) VIRTUAL,

    FOREIGN KEY (category_id) 
        REFERENCES device_categories(category_id)
        ON DELETE RESTRICT    -- Cannot delete category if devices exist
        ON UPDATE CASCADE     -- If category_id changes, update devices
) STRICT;

-- 4. WITHOUT ROWID Table
-- --------------------------------------------------------------------------------------
-- Demonstrates: Optimization for high-volume data where PK is the data.
-- Saves B-Tree space by removing the hidden 'rowid'.
CREATE TABLE IF NOT EXISTS sensor_readings (
    device_id INTEGER NOT NULL,
    recorded_at TEXT NOT NULL,
    
    raw_value REAL NOT NULL,
    unit TEXT CHECK(unit IN ('C', 'F', 'PSI', 'BAR')),

    -- GENERATED COLUMN (STORED): Calculated on write and saved to disk.
    -- useful for heavy math you don't want to repeat on every SELECT.
    normalized_value REAL GENERATED ALWAYS AS (
        CASE 
            WHEN unit = 'F' THEN (raw_value - 32) * 5 / 9
            WHEN unit = 'PSI' THEN raw_value * 0.0689476
            ELSE raw_value
        END
    ) STORED,

    -- Composite Primary Key is required for WITHOUT ROWID
    PRIMARY KEY (device_id, recorded_at),
    
    FOREIGN KEY (device_id) 
        REFERENCES devices(device_id) 
        ON DELETE CASCADE -- If device is deleted, nuke all its readings
) WITHOUT ROWID;

-- 5. Virtual Table (Full Text Search - FTS5)
-- --------------------------------------------------------------------------------------
-- Demonstrates: Search engine capabilities for logs/documents.
CREATE VIRTUAL TABLE IF NOT EXISTS maintenance_logs USING fts5(
    technician_name,
    report_body,
    severity UNINDEXED -- Store data but don't add to the search index
);

-- 6. Indexes (Advanced)
-- --------------------------------------------------------------------------------------

-- Partial Index: Only indexes active devices (Save space/Time)
CREATE INDEX IF NOT EXISTS idx_active_devices 
ON devices(category_id) 
WHERE is_active = 1;

-- Index on Expression: Indexes the result of a function
-- useful for querying inside the JSON blob efficiently
CREATE INDEX IF NOT EXISTS idx_device_firmware 
ON devices(json_extract(settings_json, '$.firmware_version'));

-- 7. Views
-- --------------------------------------------------------------------------------------
CREATE VIEW IF NOT EXISTS v_critical_devices AS
SELECT 
    d.serial_number,
    c.name as type,
    d.next_service_due,
    (SELECT COUNT(*) FROM maintenance_logs m WHERE m.report_body MATCH d.serial_number) as incident_count
FROM devices d
JOIN device_categories c ON d.category_id = c.category_id
WHERE d.is_active = 1;

-- 8. Triggers (Complex Logic)
-- --------------------------------------------------------------------------------------

-- Audit Table for Triggers
CREATE TABLE IF NOT EXISTS audit_trail (
    audit_id INTEGER PRIMARY KEY,
    table_name TEXT,
    operation TEXT,
    old_val TEXT,
    new_val TEXT,
    timestamp TEXT DEFAULT CURRENT_TIMESTAMP
);

-- UPDATE TRIGGER with Condition (WHEN)
CREATE TRIGGER IF NOT EXISTS trg_audit_device_update
AFTER UPDATE OF settings_json ON devices
FOR EACH ROW
WHEN OLD.settings_json != NEW.settings_json
BEGIN
    INSERT INTO audit_trail (table_name, operation, old_val, new_val)
    VALUES ('devices', 'UPDATE_SETTINGS', OLD.settings_json, NEW.settings_json);
END;

-- BEFORE INSERT TRIGGER (Data Validation/Abort)
CREATE TRIGGER IF NOT EXISTS trg_prevent_future_readings
BEFORE INSERT ON sensor_readings
BEGIN
    SELECT CASE
        WHEN NEW.recorded_at > datetime('now', '+1 minute') THEN
            RAISE(ABORT, 'Cannot insert sensor readings from the future!')
    END;
END;

-- INSTEAD OF TRIGGER (Make a View Updatable)
-- Allows you to run an UPDATE statement on the VIEW, which redirects to the base table.
CREATE TRIGGER IF NOT EXISTS trg_update_view_service
INSTEAD OF UPDATE ON v_critical_devices
BEGIN
    UPDATE devices 
    SET last_service_date = date('now')
    WHERE serial_number = OLD.serial_number;
END;

COMMIT;