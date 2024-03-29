CREATE TABLE IF NOT EXISTS METADATA (
    COMMITID TEXT,
    DID TEXT,
    CID TEXT,
    DATAID TEXT,
    ALIAS TEXT,
    PLAT TEXT,
    VER TEXT,
    SIZE INT,
    EXPIRATION INT,
    READER TEXT,
    WRITER TEXT,
    PRIMARY KEY(COMMITID)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS index_metadata_did_alias_ver on METADATA(DID);
CREATE INDEX IF NOT EXISTS index_metadata_did_dataid_ver_cid on METADATA(DID);
CREATE INDEX IF NOT EXISTS index_metadata_did_expiration on METADATA(DID);
CREATE INDEX IF NOT EXISTS index_metadata_did_size on METADATA(DID);
CREATE INDEX IF NOT EXISTS index_metadata_plat_owner on METADATA(PLAT);
