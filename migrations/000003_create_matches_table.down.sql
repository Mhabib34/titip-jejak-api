-- Hapus index dulu (opsional, tapi best practice)
DROP INDEX IF EXISTS idx_matches_pending_notify;
DROP INDEX IF EXISTS idx_matches_missing_report;
DROP INDEX IF EXISTS idx_matches_found_report;

-- Hapus table
DROP TABLE IF EXISTS matches;