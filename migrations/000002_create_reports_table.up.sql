CREATE TYPE report_type   AS ENUM ('found', 'missing');
CREATE TYPE report_gender AS ENUM ('male', 'female', 'unknown');
CREATE TYPE report_status AS ENUM ('active', 'resolved');

CREATE TABLE reports (
                         id                 UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
                         reporter_id        UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                         type               report_type   NOT NULL,
                         name               VARCHAR(255),                          -- nullable kalau tidak diketahui
                         gender             report_gender NOT NULL DEFAULT 'unknown',
                         estimated_age      INT           CHECK (estimated_age >= 0 AND estimated_age <= 120),
                         photo_url          VARCHAR(500),
                         description        TEXT          NOT NULL,
                         last_seen_location VARCHAR(500)  NOT NULL,
                         city               VARCHAR(100)  NOT NULL,
                         province           VARCHAR(100)  NOT NULL,
                         latitude           DOUBLE PRECISION,
                         longitude          DOUBLE PRECISION,
                         status             report_status NOT NULL DEFAULT 'active',
                         created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
                         updated_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- Index untuk filter & search
CREATE INDEX idx_reports_status   ON reports(status);
CREATE INDEX idx_reports_type     ON reports(type);
CREATE INDEX idx_reports_city     ON reports(city);
CREATE INDEX idx_reports_gender   ON reports(gender);
CREATE INDEX idx_reports_reporter ON reports(reporter_id);

-- Index untuk map view (hanya laporan aktif yang punya koordinat)
CREATE INDEX idx_reports_map ON reports(status, latitude, longitude)
    WHERE status = 'active' AND latitude IS NOT NULL AND longitude IS NOT NULL;

CREATE TRIGGER trigger_reports_updated_at
    BEFORE UPDATE ON reports
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();