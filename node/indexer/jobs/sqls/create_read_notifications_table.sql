CREATE TABLE IF NOT EXISTS READ_NOTIFICATIONS (
                                                  COMMITID      TEXT,
                                                  DATAID        TEXT,
                                                  ALIAS         TEXT,
                                                  TIME          INTEGER,
                                                  OWNER         TEXT,
                                                  MESSAGETYPE   INTEGER,
                                                  PRIMARY KEY(COMMITID)
    ) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS index_read_notifications_time ON READ_NOTIFICATIONS(TIME);
CREATE INDEX IF NOT EXISTS index_read_notifications_owner ON READ_NOTIFICATIONS(OWNER);
CREATE INDEX IF NOT EXISTS index_read_notifications_messagetype ON READ_NOTIFICATIONS(MESSAGETYPE);
CREATE INDEX IF NOT EXISTS index_read_notifications_data_id ON READ_NOTIFICATIONS(DATAID);
CREATE INDEX IF NOT EXISTS index_read_notifications_commit_id ON READ_NOTIFICATIONS(COMMITID);
