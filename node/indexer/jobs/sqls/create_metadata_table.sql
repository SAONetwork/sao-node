CREATE TABLE IF NOT EXISTS METADATA (
                                        dataId TEXT PRIMARY KEY,
                                        owner TEXT,
                                        alias TEXT,
                                        groupId TEXT,
                                        orderId TEXT,
                                        tags TEXT,
                                        cid TEXT,
                                        commits TEXT,
                                        extendInfo TEXT,
                                        `updateAt` BOOLEAN,
                                        `commitId` TEXT,
                                        rule TEXT,
                                        duration INTEGER,
                                        createdAt INTEGER,
                                        readonlyDids TEXT,
                                        readwriteDids TEXT,
                                        status INTEGER,
                                        orders TEXT
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS index_metadata_data_id on METADATA(dataId);
CREATE INDEX IF NOT EXISTS index_metadata_owner on METADATA(owner);
CREATE INDEX IF NOT EXISTS index_metadata_status on METADATA(status);
CREATE INDEX IF NOT EXISTS index_metadata_commit on METADATA(`commitId`);
CREATE INDEX IF NOT EXISTS index_metadata_order_id on METADATA(orderId);