CREATE TABLE messages (
        id BIGSERIAL PRIMARY KEY,
        msg JSONB NOT NULL
);
CREATE INDEX idxmsg ON messages USING GIN (msg);

CREATE TABLE brokers (
        username TEXT,
        password TEXT,
        broker TEXT,
        topic TEXT,
        PRIMARY KEY (username, broker)
);