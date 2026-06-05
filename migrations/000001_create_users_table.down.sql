-- Hapus trigger dulu
DROP TRIGGER IF EXISTS trigger_users_updated_at ON users;

-- Hapus function
DROP FUNCTION IF EXISTS update_updated_at_column;

-- Hapus index (opsional, karena ikut kehapus kalau table di-drop)
DROP INDEX IF EXISTS idx_users_email;

-- Hapus table
DROP TABLE IF EXISTS users;

-- Hapus enum
DROP TYPE IF EXISTS user_role;

-- (Opsional) Hapus extension
DROP EXTENSION IF EXISTS "pgcrypto";