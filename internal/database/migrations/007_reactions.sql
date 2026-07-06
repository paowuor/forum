CREATE TABLE IF NOT EXISTS reactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    target_type TEXT NOT NULL CHECK (target_type IN ('post','comment')),
    target_id INTEGER NOT NULL,
    value INTEGER NOT NULL CHECK (value IN (1,-1)),
    UNIQUE (user_id, target_type, target_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);