CREATE TABLE IF NOT EXISTS VERSE_COMMENT_LIKE (
                                                   COMMITID    TEXT,
                                                   DATAID      TEXT,
                                                   ALIAS       TEXT,
                                                   CREATEDAT   INTEGER,
                                                   UPDATEDAT   INTEGER,
                                                   COMMENTID   TEXT,
                                                   STATUS      INTEGER,
                                                   OWNER       TEXT,
                                                   PRIMARY KEY(COMMITID)
    ) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS index_verse_comment_like_comment_id on VERSE_COMMENT_LIKE(COMMENTID);
CREATE INDEX IF NOT EXISTS index_verse_comment_like_status on VERSE_COMMENT_LIKE(STATUS);
CREATE INDEX IF NOT EXISTS index_verse_comment_like_owner on VERSE_COMMENT_LIKE(OWNER);
CREATE INDEX IF NOT EXISTS index_verse_comment_like_data_id on VERSE_COMMENT_LIKE(DATAID);
CREATE INDEX IF NOT EXISTS index_verse_comment_like_commit_id on VERSE_COMMENT_LIKE(COMMITID);
