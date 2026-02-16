DROP INDEX IF EXISTS idx_group_messages_group_created;
DROP INDEX IF EXISTS idx_private_messages_from_to;
DROP INDEX IF EXISTS idx_public_messages_user_id;
DROP INDEX IF EXISTS idx_sessions_expires_at;
DROP INDEX IF EXISTS idx_private_messages_to_user;
DROP INDEX IF EXISTS idx_group_messages_user_id;
DROP INDEX IF EXISTS idx_group_messages_group_id;

DROP TABLE IF EXISTS group_messages;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS private_messages;
DROP TABLE IF EXISTS public_messages;
DROP TABLE IF EXISTS users;