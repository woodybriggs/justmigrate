CREATE TABLE orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id),
    total_amount REAL,
    shipping_address TEXT NOT NULL, 
    CONSTRAINT valid_total CHECK (total_amount >= 0)
);

CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    first_name TEXT NOT NULL, 
    email TEXT,
    phone TEXT
);

CREATE TABLE legacy_audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    log_id INTEGER NOT NULL,
    action TEXT NOT NULL,
    timestamp TEXT DEFAULT CURRENT_TIMESTAMP
);
