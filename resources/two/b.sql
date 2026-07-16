CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    -- DROP COLUMN CONSTRAINT: Removed 'NOT NULL' from first_name
    first_name TEXT, 
    -- NEW COLUMN CONSTRAINT: Added 'UNIQUE NOT NULL' to email
    email TEXT UNIQUE NOT NULL, 
    -- RENAME COLUMN: Renamed 'phone' to 'phone_number'
    phone_number TEXT 
);

CREATE TABLE orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    -- DROP TABLE CONSTRAINT: Removed 'CONSTRAINT valid_total CHECK (total_amount >= 0)'
    total_amount REAL,
    -- DROP COLUMN: Removed 'shipping_address' (assumed moved to a normalized addresses table)
    -- NEW COLUMN: Added 'status'
    status TEXT DEFAULT 'cart',
    -- NEW TABLE CONSTRAINT: Added a composite check constraint
    CONSTRAINT valid_checkout CHECK ((status = 'cart') OR (total_amount > 0))
);

-- DROP TABLE: 'legacy_audit_logs' has been completely removed from the schema

-- NEW TABLE: Added 'product_reviews'
CREATE TABLE product_reviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    rating INTEGER CHECK (rating BETWEEN 1 AND 5),
    review_text TEXT
);
