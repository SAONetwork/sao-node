CREATE TABLE IF NOT EXISTS VERSE_LIKE (
                                                   COMMITID    TEXT,
                                                   DATAID      TEXT,
                                                   ALIAS       TEXT,
                                                   CREATEDAT   INTEGER,
                                                   UPDATEDAT   INTEGER,
                                                   VERSEID   TEXT,
                                                   STATUS      INTEGER,
                                                   OWNER       TEXT,
                                                   PRIMARY KEY(COMMITID)
    ) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS index_verse_like_verse_id on VERSE_LIKE(VERSEID);
CREATE INDEX IF NOT EXISTS index_verse_like_status on VERSE_LIKE(STATUS);
CREATE INDEX IF NOT EXISTS index_verse_like_owner on VERSE_LIKE(OWNER);
CREATE INDEX IF NOT EXISTS index_verse_like_data_id on VERSE_LIKE(DATAID);
CREATE INDEX IF NOT EXISTS index_verse_like_commit_id on VERSE_LIKE(COMMITID);