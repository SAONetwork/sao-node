CREATE TABLE IF NOT EXISTS USER_FOLLOWING (
                                              COMMITID    TEXT,
                                              DATAID      TEXT,
                                              ALIAS       TEXT,
                                              CREATEDAT   INTEGER,
                                              UPDATEDAT   INTEGER,
                                              EXPIREDAT   INTEGER,
                                              FOLLOWER    TEXT,
                                              `FOLLOWING`   TEXT,
                                              STATUS      INTEGER,
                                              PRIMARY KEY(COMMITID)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS index_user_following_follower on user_following(FOLLOWER);
CREATE INDEX IF NOT EXISTS index_user_following_following on user_following(`FOLLOWING`);
CREATE INDEX IF NOT EXISTS index_user_following_data_id on user_following(DATAID);
CREATE INDEX IF NOT EXISTS index_user_following_commit_id on user_following(COMMITID);