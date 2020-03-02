CREATE TABLE public.message (
        id bigserial PRIMARY KEY,
        subscribe_id bigserial,
        topic text NOT NULL,
        body jsonb NOT NULL
);
CREATE INDEX idx_subid_topic ON messages (subscribe_id, topic);
CREATE INDEX idxbody ON messages USING GIN (body);

CREATE TABLE public.subscribe (
        id char(36) PRIMARY KEY,
        username text,
        password text,
        brokers text,
        topics text,
        qos char(1) NOT NULL
);