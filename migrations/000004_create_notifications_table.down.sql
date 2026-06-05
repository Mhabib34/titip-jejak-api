-- Hapus index dulu (opsional tapi best practice)
DROP INDEX IF EXISTS idx_notifications_unread;
DROP INDEX IF EXISTS idx_notifications_user;

-- Hapus table
DROP TABLE IF EXISTS notifications;