CREATE TABLE IF NOT EXISTS VERSE (
     COMMITID    TEXT,
     DATAID      TEXT,
     ALIAS  TEXT,
     CREATEDAT   INTEGER,
     FILEIDS     TEXT,
     OWNER       TEXT,
     PRICE       REAL,
     DIGEST      TEXT,
     `SCOPE`       INTEGER,
     STATUS      INTEGER,
     NFTTOKENID  TEXT,
     PRIMARY KEY(COMMITID)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS index_verses_owner on verse(OWNER);
CREATE INDEX IF NOT EXISTS index_verses_scope on verse(`SCOPE`);
CREATE INDEX IF NOT EXISTS index_verses_status on verse(STATUS);
CREATE INDEX IF NOT EXISTS index_verses_data_id on verse(DATAID);
CREATE INDEX IF NOT EXISTS index_verses_commit_id on verse(COMMITID);