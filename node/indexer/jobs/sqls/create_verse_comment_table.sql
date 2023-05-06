CREATE TABLE IF NOT EXISTS VERSE_COMMENT (
                                             COMMITID    TEXT,
                                             DATAID      TEXT,
                                             ALIAS       TEXT,
                                             CREATEDAT INTEGER,
                                             UPDATEDAT INTEGER,
                                             COMMENT    TEXT,
                                             PARENTID  TEXT,
                                             VERSEID   TEXT,
                                             OWNER      TEXT,
                                             PRIMARY KEY(COMMITID)
    ) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS index_verse_comment_parent_id on VERSE_COMMENT(PARENTID);
CREATE INDEX IF NOT EXISTS index_verse_comment_verse_id on VERSE_COMMENT(VERSEID);
CREATE INDEX IF NOT EXISTS index_verse_comment_owner on VERSE_COMMENT(OWNER);
CREATE INDEX IF NOT EXISTS index_verse_comment_data_id on VERSE_COMMENT(DATAID);
CREATE INDEX IF NOT EXISTS index_verse_comment_commit_id on VERSE_COMMENT(COMMITID);
