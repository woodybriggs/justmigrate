DROP TABLE [legacy_audit_logs];

CREATE TABLE [product_reviews] (
    [id] INTEGER PRIMARY KEY,
    [user_id] INTEGER NOT NULL,
    [product_id] INTEGER NOT NULL,
    [rating] INTEGER CHECK ([rating] BETWEEN 1 AND 5),
    [review_text] TEXT
);

CREATE TABLE [tmpnew_orders] (
    [id] INTEGER PRIMARY KEY,
    [user_id] INTEGER NOT NULL,
    [total_amount] REAL,
    [status] TEXT DEFAULT 'cart',
    CONSTRAINT [valid_checkout] CHECK ([status] = 'cart' OR [total_amount] > 0)
);

INSERT INTO [tmpnew_orders] (
    [status],
    [id],
    [user_id],
    [total_amount]
) SELECT 'cart',
    [id],
    [user_id],
    [total_amount]
FROM [orders];

DROP TABLE [orders];

ALTER TABLE [tmpnew_orders] RENAME TO [orders];

CREATE TABLE [tmpnew_users] (
    [id] INTEGER PRIMARY KEY,
    [username] TEXT UNIQUE,
    [first_name] TEXT,
    [email] TEXT NOT NULL,
    [phone_number] TEXT
);

INSERT INTO [tmpnew_users] (
    [username],
    [first_name],
    [email],
    [phone_number],
    [id]
) SELECT [username],
    [first_name],
    [email],
    NULL,
    [id]
FROM [users];

DROP TABLE [users];

ALTER TABLE [tmpnew_users] RENAME TO [users];