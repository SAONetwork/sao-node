CREATE TABLE IF NOT EXISTS FILE_CONTENT (
    COMMITID        TEXT,
    DATAID          TEXT,
    ALIAS           TEXT,
    CREATEDAT       INTEGER,
    OWNER           TEXT,
    CONTENTPATH     TEXT,
    PRIMARY KEY(COMMITID)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS index_file_content_owner on file_content(OWNER);
CREATE INDEX IF NOT EXISTS index_file_content_data_id on file_content(DATAID);