CREATE TABLE matches (
                         id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
                         found_report_id   UUID        NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
                         missing_report_id UUID        NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
                         score             INT         NOT NULL CHECK (score >= 0 AND score <= 100),
                         notified          BOOLEAN     NOT NULL DEFAULT FALSE,
                         created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Satu pasang laporan hanya bisa match sekali
                         CONSTRAINT uq_match_pair UNIQUE (found_report_id, missing_report_id)
);

CREATE INDEX idx_matches_found_report   ON matches(found_report_id);
CREATE INDEX idx_matches_missing_report ON matches(missing_report_id);

-- Cron job pakai index ini untuk cari yang belum dinotifikasi
CREATE INDEX idx_matches_pending_notify ON matches(notified, score)
    WHERE notified = FALSE;