CREATE TABLE IF NOT EXISTS LISTING_INFO (
                                            COMMITID TEXT,
                                            DATAID TEXT,
                                            ALIAS  TEXT,
                                            PRICE TEXT,
                                            TOKENID TEXT,
                                            ITEMDATAID TEXT,
                                            CHAINTYPE TEXT,
                                            TIME INT,
                                            PRIMARY KEY(COMMITID)
    ) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS index_listing_info_data_id on LISTING_INFO(DATAID);
CREATE INDEX IF NOT EXISTS index_listing_info_id on LISTING_INFO(COMMITID);
CREATE INDEX IF NOT EXISTS index_listing_info_token_id on LISTING_INFO(TOKENID);
CREATE INDEX IF NOT EXISTS index_listing_info_item_data_id on LISTING_INFO(ITEMDATAID);