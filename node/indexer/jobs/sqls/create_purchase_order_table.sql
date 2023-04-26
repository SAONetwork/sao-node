CREATE TABLE IF NOT EXISTS PURCHASE_ORDER (
                                              COMMITID    TEXT,
                                              DATAID      TEXT,
                                              ALIAS       TEXT,
                                              ORDERID     INTEGER,
                                              ITEMDATAID TEXT,
                                              BUYERDATAID TEXT,
                                              ORDERTXHASH TEXT,
                                              CHAINTYPE   TEXT,
                                              PRICE       TEXT,
                                              TIME        INTEGER,
                                              TYPE        INTEGER,
                                              EXPIRETIME  INTEGER,
                                              PRIMARY KEY (COMMITID)
    ) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS index_purchase_order_data_id ON PURCHASE_ORDER(DATAID);
CREATE INDEX IF NOT EXISTS index_purchase_order_buyer_data_id ON PURCHASE_ORDER(BUYERDATAID);
CREATE INDEX IF NOT EXISTS index_purchase_order_item_data_id ON PURCHASE_ORDER(ITEMDATAID);
