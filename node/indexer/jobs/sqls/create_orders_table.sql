CREATE TABLE IF NOT EXISTS ORDERS (
                                      creator TEXT,
                                      owner TEXT,
                                      id TEXT PRIMARY KEY,
                                      provider TEXT,
                                      cid TEXT,
                                      duration TEXT,
                                      status INTEGER,
                                      replica INTEGER,
                                      denom TEXT,
                                      amount TEXT,
                                      `size` TEXT,
                                      operation INTEGER,
                                      createdAt TEXT,
                                      timeout TEXT,
                                      dataId TEXT,
                                      `commitId` TEXT,
                                      unitPrice TEXT
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS index_order_id on ORDERS(id);
CREATE INDEX IF NOT EXISTS index_order_owner on ORDERS(owner);
CREATE INDEX IF NOT EXISTS index_order_status on ORDERS(status);
CREATE INDEX IF NOT EXISTS index_order_commit on ORDERS(`commitId`);