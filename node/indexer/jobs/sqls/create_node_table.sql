CREATE TABLE IF NOT EXISTS NODE (
                                    Creator          TEXT,
                                    Peer             TEXT,
                                    Reputation       REAL,
                                    Status           INTEGER,
                                    LastAliveHeight  INTEGER,
                                    TxAddresses      TEXT,
                                    `Role`             INTEGER,
                                    Validator        TEXT,
                                    IsGateway       INTEGER,
                                    IsSP           INTEGER,
                                    IsIndexer      INTEGER,
                                    IsAlive        INTEGER,
                                    IPAddress      TEXT,
                                    LastAliveTime INTEGER,
                                    `Name`          TEXT,
                                    Details         TEXT,
                                    `Identity`       TEXT,
                                    SecurityContact TEXT,
                                    Website         TEXT,
                                    PRIMARY KEY (Creator)
    ) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS index_node_isgateway ON NODE (IsGateway);
CREATE INDEX IF NOT EXISTS index_node_status on NODE(status);
CREATE INDEX IF NOT EXISTS index_node_creator ON NODE(creator);
CREATE INDEX IF NOT EXISTS index_node_peer ON NODE(peer);
CREATE INDEX IF NOT EXISTS index_node_isalive ON NODE(IsAlive);
