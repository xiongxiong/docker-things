CREATE TABLE message (
        id BIGSERIAL PRIMARY KEY,
        msg JSONB NOT NULL
);
CREATE INDEX idxmsg ON message USING GIN (msg);