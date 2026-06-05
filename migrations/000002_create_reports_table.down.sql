-- Hapus trigger dulu
DROP TRIGGER IF EXISTS trigger_reports_updated_at ON reports;

-- Hapus index (opsional, tapi lebih clean)
DROP INDEX IF EXISTS idx_reports_map;
DROP INDEX IF EXISTS idx_reports_reporter;
DROP INDEX IF EXISTS idx_reports_gender;
DROP INDEX IF EXISTS idx_reports_city;
DROP INDEX IF EXISTS idx_reports_type;
DROP INDEX IF EXISTS idx_reports_status;

-- Hapus table
DROP TABLE IF EXISTS reports;

-- Hapus enum
DROP TYPE IF EXISTS report_status;
DROP TYPE IF EXISTS report_gender;
DROP TYPE IF EXISTS report_type;