CREATE TABLE notifications (
                               id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
                               user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                               report_id  UUID        REFERENCES reports(id) ON DELETE SET NULL,
                               match_id   UUID        REFERENCES matches(id) ON DELETE SET NULL,
                               message    TEXT        NOT NULL,
                               is_read    BOOLEAN     NOT NULL DEFAULT FALSE,
                               created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Load notifikasi per user, terbaru duluan
CREATE INDEX idx_notifications_user ON notifications(user_id, created_at DESC);

-- Hitung badge notifikasi belum dibaca
CREATE INDEX idx_notifications_unread ON notifications(user_id, is_read)
    WHERE is_read = FALSE;