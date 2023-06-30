CREATE TABLE IF NOT EXISTS FILE_INFO (
                                         COMMITID     TEXT,
                                         DATAID       TEXT,
                                         ALIAS        TEXT,
                                         CREATEDAT    INTEGER,
                                         FILEDATAID   TEXT,
                                         CONTENTTYPE  TEXT,
                                         OWNER        TEXT,
                                         FILENAME     TEXT,
                                         FILECATEGORY TEXT,
                                         EXTENDINFO  TEXT,
                                         THUMBNAILDATAID TEXT
);

CREATE INDEX IF NOT EXISTS index_file_info_owner on file_info(OWNER);
CREATE INDEX IF NOT EXISTS index_file_info_data_id on file_info(DATAID);
CREATE INDEX IF NOT EXISTS index_file_info_commit_id on file_info(COMMITID);