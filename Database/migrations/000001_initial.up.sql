CREATE TABLE IF NOT EXISTS users (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    username               TEXT UNIQUE NOT NULL,
    password_hash          TEXT NOT NULL,
    signing_public_key     TEXT UNIQUE NOT NULL,
    encryption_public_key  TEXT UNIQUE NOT NULL,
    created_at             TEXT DEFAULT (CURRENT_TIMESTAMP)
) STRICT;

CREATE TABLE IF NOT EXISTS public_messages (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL,
    content      TEXT NOT NULL,
    ec_signature TEXT NOT NULL,
    created_at   TEXT DEFAULT (CURRENT_TIMESTAMP),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) STRICT;

CREATE TABLE IF NOT EXISTS private_messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    from_user_id    INTEGER NOT NULL,
    to_user_id      INTEGER NOT NULL,
    encrypted_content TEXT NOT NULL,
    nonce           TEXT NOT NULL,
    ec_signature    TEXT NOT NULL,
    created_at      TEXT DEFAULT (CURRENT_TIMESTAMP),
    read            INTEGER DEFAULT 0,
    FOREIGN KEY (from_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (to_user_id)   REFERENCES users(id) ON DELETE CASCADE
) STRICT;

CREATE TABLE IF NOT EXISTS user_sessions (
    id         TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL,
    expires_at TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) STRICT;

CREATE TABLE IF NOT EXISTS groups (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    group_name TEXT NOT NULL,
    created_by INTEGER NOT NULL,
    created_at TEXT DEFAULT (CURRENT_TIMESTAMP),
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
) STRICT;

CREATE TABLE IF NOT EXISTS group_members (
    group_id           INTEGER NOT NULL,
    user_id            INTEGER NOT NULL,
    encrypted_group_key TEXT NOT NULL,
    joined_at          TEXT DEFAULT (CURRENT_TIMESTAMP),
    PRIMARY KEY (group_id, user_id),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id)  REFERENCES users(id) ON DELETE CASCADE
) STRICT;

CREATE TABLE IF NOT EXISTS group_messages (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id      INTEGER NOT NULL,
    user_id       INTEGER NOT NULL,
    encrypted_content TEXT NOT NULL,
    nonce         TEXT NOT NULL,
    ec_signature  TEXT NOT NULL,
    created_at    TEXT DEFAULT (CURRENT_TIMESTAMP),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id)  REFERENCES users(id) ON DELETE CASCADE
) STRICT;

CREATE INDEX IF NOT EXISTS idx_group_messages_group_id    ON group_messages(group_id);
CREATE INDEX IF NOT EXISTS idx_group_messages_user_id     ON group_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_private_messages_to_user   ON private_messages(to_user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at        ON user_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_public_messages_user_id    ON public_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_private_messages_from_to   ON private_messages(from_user_id, to_user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_group_messages_group_created ON group_messages(group_id, created_at);